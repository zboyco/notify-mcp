package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/zboyco/notify-mcp/internal/config"
	"github.com/zboyco/notify-mcp/internal/mcp"
)

type stringFlag struct {
	value string
	isSet bool
}

func (f *stringFlag) String() string {
	return f.value
}

func (f *stringFlag) Set(val string) error {
	f.value = val
	f.isSet = true
	return nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			if err := runConfig(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "配置命令失败: %v\n", err)
				os.Exit(1)
			}
			return
		case "-h", "--help":
			printRootUsage(os.Args[0])
			return
		}
	}

	if err := runServer(); err != nil {
		fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		method                string
		apiURL, chatID, token string
		remove                bool
		showHelp              bool
		messageFlag           stringFlag
	)
	fs.StringVar(&method, "method", "", "要配置的通知方式，例如 telegram 或 os")
	fs.StringVar(&apiURL, "api-url", "", "Telegram API基础地址，默认为 https://api.telegram.org")
	fs.StringVar(&chatID, "chat-id", "", "Telegram Chat ID")
	fs.StringVar(&token, "token", "", "Telegram Bot Token")
	fs.Var(&messageFlag, "message", "通知内容，默认为 '即将进行汇报，请注意查看...'")
	fs.BoolVar(&remove, "remove", false, "移除指定的通知方式")
	fs.BoolVar(&showHelp, "h", false, "显示帮助信息")
	fs.BoolVar(&showHelp, "help", false, "显示帮助信息")

	fs.Usage = func() {
		printConfigUsage(os.Args[0])
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printConfigUsage(os.Args[0])
			return nil
		}
		return err
	}

	if showHelp {
		printConfigUsage(os.Args[0])
		return nil
	}

	methodChangeRequested := method != "" || apiURL != "" || chatID != "" || token != "" || remove
	updateRequested := methodChangeRequested || messageFlag.isSet
	if !updateRequested {
		return showCurrentConfig()
	}

	if methodChangeRequested && method == "" {
		return errors.New("更新通知配置时必须通过 --method 指定通知方式")
	}

	settings := config.Settings{}
	if existing, err := config.Load(); err == nil {
		settings = existing
	} else if !errors.Is(err, config.ErrNotConfigured) {
		return err
	} else if remove {
		return errors.New("当前尚未配置任何通知方式，无法移除")
	}

	if methodChangeRequested {
		if remove {
			if apiURL != "" || chatID != "" || token != "" {
				return errors.New("移除通知方式时无需提供 --api-url/--chat-id/--token 参数")
			}
			var removed bool
			settings.Methods, removed = removeMethod(settings.Methods, config.MethodType(method))
			if !removed {
				return fmt.Errorf("通知方式 %s 尚未配置", method)
			}
		} else {
			switch config.MethodType(method) {
			case config.MethodTelegram:
				if chatID == "" || token == "" {
					return errors.New("更新 Telegram 配置时必须提供 --chat-id, --token，可选 --api-url")
				}
				if apiURL == "" {
					apiURL = config.DefaultTelegramAPIBaseURL
				}
				tgCfg := config.TelegramConfig{
					APIBaseURL: apiURL,
					ChatID:     chatID,
					Token:      token,
				}
				method, err := config.NewTelegramMethod(tgCfg)
				if err != nil {
					return err
				}
				settings.Methods = upsertMethod(settings.Methods, method)
			case config.MethodOS:
				if apiURL != "" || chatID != "" || token != "" {
					return errors.New("操作系统通知无需 --api-url/--chat-id/--token 参数")
				}
				method, err := config.NewOSMethod()
				if err != nil {
					return err
				}
				settings.Methods = upsertMethod(settings.Methods, method)
			default:
				return fmt.Errorf("不支持的通知方式: %s", method)
			}
		}
	}

	if messageFlag.isSet {
		settings.NotificationMessage = messageFlag.value
	}

	if len(settings.Methods) == 0 {
		return errors.New("请至少指定一种通知方式（例如 Telegram 或 os）")
	}

	if err := config.Save(settings); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "配置已保存。")
	return nil
}

func showCurrentConfig() error {
	settings, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrNotConfigured) {
			return fmt.Errorf("尚未配置通知方式，请运行 `%s config --method telegram --chat-id ... --token ... [--api-url ...]` 或 `--method os`", os.Args[0])
		}
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("格式化配置失败: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runServer() error {
	settings, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrNotConfigured) {
			return fmt.Errorf("尚未配置通知方式，请运行 `%s config --method telegram --chat-id ... --token ... [--api-url ...]` 或添加 `--method os`", os.Args[0])
		}
		return err
	}

	if len(settings.Methods) == 0 {
		return fmt.Errorf("当前通知配置为空，请运行 `%s config --method ...` 添加至少一种通知方式", os.Args[0])
	}

	logger := log.New(os.Stderr, "[notify-mcp-mcp] ", log.LstdFlags)
	logger.Printf("可执行文件路径: %s", os.Args[0])
	if cfgPath, pathErr := config.Path(); pathErr == nil {
		logger.Printf("加载配置文件: %s", cfgPath)
	} else {
		logger.Printf("无法确定配置路径: %v", pathErr)
	}
	var methodTypes []string
	for _, method := range settings.Methods {
		methodTypes = append(methodTypes, string(method.Type))
	}
	logger.Printf("配置校验通过，已启用通知方式: %v", methodTypes)
	server := mcp.NewServer(settings, logger)
	logger.Println("notify-mcp 服务器启动，等待 Claude Code 连接 ...")
	return server.Serve()
}

func printConfigUsage(program string) {
	name := filepath.Base(program)
	fmt.Fprintf(os.Stdout, `用法:
  %s config
      显示当前配置内容。

  %s config --method <method> [其它参数]
      根据通知方式更新或移除配置。method 取值：telegram, os

参数说明:
  --method    要配置的通知方式（telegram / os）
  --remove    移除指定通知方式
  --api-url   Telegram API基础地址，默认为 https://api.telegram.org
  --chat-id   Telegram Chat ID
  --token     Telegram Bot Token
  --message   通知内容文案
`, name, name)
}

func printRootUsage(program string) {
	name := filepath.Base(program)
	fmt.Fprintf(os.Stdout, `用法:
  %s config [参数]
      查看或更新通知配置（详见 %s config -h）。

  %s
      启动 notify-mcp 服务，需提前完成配置。
`, name, name, name)
}

func upsertMethod(methods []config.Method, method config.Method) []config.Method {
	for i, item := range methods {
		if item.Type == method.Type {
			methods[i] = method
			return methods
		}
	}
	return append(methods, method)
}

func removeMethod(methods []config.Method, methodType config.MethodType) ([]config.Method, bool) {
	for i, item := range methods {
		if item.Type == methodType {
			return append(methods[:i], methods[i+1:]...), true
		}
	}
	return methods, false
}
