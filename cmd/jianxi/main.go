package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"jianxi/internal/blog"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
func run() error {
	command := "serve"
	args := os.Args[1:]
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	if command == "mcp" {
		flags := flag.NewFlagSet("mcp", flag.ContinueOnError)
		defaultRenderer := os.Getenv("JIANXI_RENDERER")
		if defaultRenderer == "" {
			executable, _ := os.Executable()
			defaultRenderer = filepath.Join(filepath.Dir(executable), "..", "scripts", "compile.mjs")
			if _, err := os.Stat(defaultRenderer); err != nil {
				defaultRenderer = "scripts/compile.mjs"
			}
		}
		renderer := flags.String("renderer", defaultRenderer, "本地固定编译器的绝对路径")
		if err := flags.Parse(args); err != nil {
			return err
		}
		return blog.RunMCP(context.Background(), *renderer)
	}
	if command == "healthcheck" {
		client := &http.Client{Timeout: 4 * time.Second}
		response, err := client.Get("http://127.0.0.1:8080/healthz")
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("healthcheck: HTTP %d", response.StatusCode)
		}
		return nil
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	data := flags.String("data", "data", "私有持久数据目录")
	var clearInitialToken *bool
	if command == "init-info" {
		clearInitialToken = flags.Bool("clear-initial-token", false, "保存密钥后移除初始化交付文件")
	}
	addr := flags.String("addr", "127.0.0.1:8080", "HTTP 监听地址")
	demo := flags.Bool("demo", false, "仅在空站导入明确标记的演示内容")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if command == "init-info" {
		route, err := os.ReadFile(filepath.Join(*data, "admin-route.txt"))
		if err != nil {
			return err
		}
		fmt.Printf("管理入口：%s", route)
		path := filepath.Join(*data, "initial-token.txt")
		if *clearInitialToken {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			fmt.Println("初始化密钥交付文件已移除。")
			return nil
		}
		token, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			fmt.Println("初始化密钥文件已移除；请使用已保存的密钥。")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("初始化管理密钥：%s", token)
		return nil
	}
	if command == "token-reset" {
		token, err := blog.RotateToken(*data)
		if err != nil {
			return err
		}
		fmt.Println(token)
		return nil
	}
	if command != "serve" {
		return fmt.Errorf("用法：jianxi serve | mcp | init-info | token-reset | healthcheck")
	}
	app, err := blog.NewApp(*data, "")
	if err != nil {
		return err
	}
	defer app.Store.Close()
	if *demo {
		if err = app.SeedDemo(context.Background()); err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go app.RunDiscovery(ctx)
	server := &http.Server{Addr: *addr, Handler: app.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 5 * time.Minute, WriteTimeout: 5 * time.Minute, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	root, _ := filepath.Abs(*data)
	log.Printf("间隙已启动：http://%s；私有数据 %s；管理入口见 admin-route.txt", *addr, root)
	if _, err = os.Stat(filepath.Join(root, "initial-token.txt")); err == nil {
		log.Printf("首次管理密钥位于 %s；请妥善保存后移除该初始化文件", filepath.Join(root, "initial-token.txt"))
	}
	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
