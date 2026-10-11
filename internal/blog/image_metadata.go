package blog

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type ImageMetadata struct {
	Format string `json:"format"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (c *Compiler) checkLocalImage(ctx context.Context, data []byte) error {
	expected, err := inspectImage(data)
	if err != nil {
		return err
	}
	result, err := c.run(ctx, map[string]any{"kind": "image", "source": base64.StdEncoding.EncodeToString(data)})
	if err != nil {
		return err
	}
	format, ok := result["format"].(string)
	width, wok := result["width"].(float64)
	height, hok := result["height"].(float64)
	if !ok || !wok || !hok || format != expected.Format || width != float64(expected.Width) || height != float64(expected.Height) {
		return fail("INVALID_IMAGE", "本地图片解码与服务端容器校验结果不一致")
	}
	return nil
}

// The MCP performs Sharp validation. The server independently checks the bytes,
// container lengths, format and pixel dimensions without a native image runtime.
func inspectImage(data []byte) (ImageMetadata, error) {
	out := ImageMetadata{}
	if len(data) < 12 || len(data) > 10<<20 {
		return out, fail("INVALID_IMAGE", "图片不合法或超过 10 MiB")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err == nil {
		out = ImageMetadata{format, config.Width, config.Height}
	} else if string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		out, err = inspectWebP(data)
	} else if string(data[4:8]) == "ftyp" {
		out, err = inspectAVIF(data)
	} else {
		return out, fail("INVALID_IMAGE", "只接受 PNG/JPEG/WebP/GIF/AVIF 图片")
	}
	if err != nil {
		return ImageMetadata{}, fail("INVALID_IMAGE", "图片容器或尺寸无效")
	}
	if out.Width <= 0 || out.Height <= 0 || out.Width > 40_000_000 || out.Height > 40_000_000 || int64(out.Width)*int64(out.Height) > 40_000_000 {
		return ImageMetadata{}, fail("INVALID_IMAGE", "图片像素超过限制")
	}
	return out, nil
}

// RIFF dimensions follow https://developers.google.com/speed/webp/docs/riff_container.
func inspectWebP(data []byte) (ImageMetadata, error) {
	out := ImageMetadata{Format: "webp"}
	invalid := fail("INVALID_IMAGE", "WebP 容器无效")
	if uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return out, invalid
	}
	imageData := false
	for offset := 12; offset < len(data); {
		if len(data)-offset < 8 {
			return out, invalid
		}
		n := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := uint64(offset) + 8 + n
		if end > uint64(len(data)) {
			return out, invalid
		}
		b := data[offset+8 : int(end)]
		switch string(data[offset : offset+4]) {
		case "VP8X":
			if len(b) != 10 {
				return out, invalid
			}
			out.Width = 1 + int(b[4]) + int(b[5])<<8 + int(b[6])<<16
			out.Height = 1 + int(b[7]) + int(b[8])<<8 + int(b[9])<<16
		case "VP8 ":
			if len(b) < 10 || b[0]&1 != 0 || !bytes.Equal(b[3:6], []byte{0x9d, 0x01, 0x2a}) {
				return out, invalid
			}
			imageData = true
			if out.Width == 0 {
				out.Width = int(binary.LittleEndian.Uint16(b[6:8]) & 0x3fff)
				out.Height = int(binary.LittleEndian.Uint16(b[8:10]) & 0x3fff)
			}
		case "VP8L":
			if len(b) < 5 || b[0] != 0x2f {
				return out, invalid
			}
			imageData = true
			bits := binary.LittleEndian.Uint32(b[1:5])
			if out.Width == 0 {
				out.Width = 1 + int(bits&0x3fff)
				out.Height = 1 + int((bits>>14)&0x3fff)
			}
		case "ANMF":
			if len(b) < 24 {
				return out, invalid
			}
			imageData = true
		}
		if n%2 == 1 {
			if end >= uint64(len(data)) || data[end] != 0 {
				return out, invalid
			}
			end++
		}
		offset = int(end)
	}
	if !imageData || out.Width == 0 || out.Height == 0 {
		return out, invalid
	}
	return out, nil
}

// AVIF brands and image spatial extents follow https://aomediacodec.github.io/av1-avif/.
func inspectAVIF(data []byte) (ImageMetadata, error) {
	out := ImageMetadata{Format: "avif"}
	brand, codec, payload := false, false, false
	count := 0
	invalid := fail("INVALID_IMAGE", "AVIF 容器无效")
	var walk func([]byte, int) error
	walk = func(b []byte, depth int) error {
		if depth > 16 {
			return invalid
		}
		for len(b) > 0 {
			if len(b) < 8 {
				return invalid
			}
			size := uint64(binary.BigEndian.Uint32(b[:4]))
			header := 8
			if size == 1 {
				if len(b) < 16 {
					return invalid
				}
				size = binary.BigEndian.Uint64(b[8:16])
				header = 16
			} else if size == 0 {
				size = uint64(len(b))
			}
			if size < uint64(header) || size > uint64(len(b)) {
				return invalid
			}
			count++
			if count > 4096 {
				return invalid
			}
			kind := string(b[4:8])
			body := b[header:int(size)]
			switch kind {
			case "ftyp":
				if depth != 0 || len(body) < 8 || (len(body)-8)%4 != 0 {
					return invalid
				}
				for i := 0; i < len(body); i += 4 {
					if i == 4 {
						continue
					}
					if string(body[i:i+4]) == "avif" || string(body[i:i+4]) == "avis" {
						brand = true
					}
				}
			case "meta":
				if len(body) < 4 {
					return invalid
				}
				if err := walk(body[4:], depth+1); err != nil {
					return err
				}
			case "iprp", "ipco":
				if err := walk(body, depth+1); err != nil {
					return err
				}
			case "ispe":
				if len(body) != 12 || !bytes.Equal(body[:4], []byte{0, 0, 0, 0}) {
					return invalid
				}
				w, h := binary.BigEndian.Uint32(body[4:8]), binary.BigEndian.Uint32(body[8:12])
				if w == 0 || h == 0 || w > 40_000_000 || h > 40_000_000 || uint64(w)*uint64(h) > 40_000_000 {
					return invalid
				}
				out.Width = max(out.Width, int(w))
				out.Height = max(out.Height, int(h))
			case "av1C":
				if len(body) < 4 || body[0] != 0x81 {
					return invalid
				}
				codec = true
			case "mdat", "idat":
				if len(body) > 0 {
					payload = true
				}
			}
			b = b[int(size):]
		}
		return nil
	}
	if err := walk(data, 0); err != nil {
		return out, err
	}
	if !brand || !codec || !payload || out.Width == 0 || out.Height == 0 {
		return out, invalid
	}
	return out, nil
}
