package blog

import (
	"encoding/csv"
	"io"
	"math"
	"strconv"
	"strings"
)

type ReportCSVInput struct {
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	ObservedAt string `json:"observedAt"`
	Definition string `json:"definition"`
	CSV        string `json:"csv"`
}

func (a *App) ImportCSV(in ReportCSVInput) (Report, error) {
	report := Report{Version: 1, Kind: in.Kind, Source: in.Source, ObservedAt: in.ObservedAt, Definition: in.Definition}
	if len(in.CSV) > 2<<20 {
		return report, fail("REPORT_TOO_LARGE", "CSV 最多 2 MiB")
	}
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(in.CSV, "\ufeff")))
	head, err := r.Read()
	if err != nil || strings.Join(head, ",") != "platform,query,url,value,evidence" {
		return report, fail("INVALID_REPORT", "CSV 表头必须为 platform,query,url,value,evidence")
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) != 5 {
			return report, fail("INVALID_REPORT", "CSV 行格式不合法")
		}
		value, err := strconv.ParseFloat(row[3], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return report, fail("INVALID_REPORT", "value 必须为数字")
		}
		report.Rows = append(report.Rows, ReportRow{Platform: row[0], Query: row[1], URL: row[2], Value: value, Evidence: row[4]})
		if len(report.Rows) > 10000 {
			return report, fail("INVALID_REPORT", "最多 10000 行")
		}
	}
	return a.ImportReport(report)
}
