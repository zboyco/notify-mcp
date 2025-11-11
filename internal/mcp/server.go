package mcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zboyco/notify-mcp/internal/config"
	"github.com/zboyco/notify-mcp/internal/osnotify"
	"github.com/zboyco/notify-mcp/internal/telegram"
)

const (
	serverName      = "notify-mcp"
	serverVersion   = "0.1.0"
	toolName        = "notify"
	taskNameParam   = "taskName"
	defaultTaskName = "当前任务"
)

// Server wraps an mcp-go server with the notify tool registered.
type Server struct {
	cfg       config.Settings
	logger    *log.Logger
	mcpServer *server.MCPServer
}

// NewServer builds a new MCP server backed by mark3labs/mcp-go.
func NewServer(cfg config.Settings, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
		server.WithLogging(),
	)

	s := &Server{
		cfg:       cfg,
		logger:    logger,
		mcpServer: mcpServer,
	}
	s.registerTools()
	return s
}

// Serve starts the stdio transport using the official SDK.
func (s *Server) Serve() error {
	return server.ServeStdio(
		s.mcpServer,
		server.WithErrorLogger(s.logger),
	)
}

func (s *Server) registerTools() {
	tool := mcp.NewTool(
		toolName,
		mcp.WithDescription("向已配置的渠道发送通知"),
		mcp.WithString(
			taskNameParam,
			mcp.Description("当前执行任务的缩略标题"),
			mcp.DefaultString(defaultTaskName),
		),
		mcp.WithTitleAnnotation("notify"),
		mcp.WithDestructiveHintAnnotation(false),
	)

	s.mcpServer.AddTool(tool, s.handleNotifyTool)
}

func (s *Server) handleNotifyTool(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	taskName := strings.TrimSpace(req.GetString(taskNameParam, defaultTaskName))
	settings, err := config.Load()
	if err != nil {
		s.logger.Printf("重新加载配置失败: %v", err)
		return mcp.NewToolResultError("读取通知配置失败"), nil
	}
	if len(settings.Methods) == 0 {
		s.logger.Println("配置中未包含通知方式")
		return mcp.NewToolResultError("未配置任何通知方式"), nil
	}
	body := settings.EffectiveNotificationMessage()
	message := fmt.Sprintf("时间：%s\n任务：%s\n%s", time.Now().Format("2006-01-02 15:04:05"), taskName, body)

	var successChannels []string
	var failedChannels []string

	for _, method := range settings.Methods {
		var err error
		switch method.Type {
		case config.MethodTelegram:
			var tgCfg config.TelegramConfig
			tgCfg, err = method.TelegramConfig()
			if err == nil {
				err = telegram.SendMessage(ctx, tgCfg, message)
			}
		case config.MethodOS:
			err = osnotify.Send(ctx, "AI通知助手", message)
		default:
			err = fmt.Errorf("未知通知方式: %s", method.Type)
		}

		if err != nil {
			s.logger.Printf("通知方式 %s 发送失败: %v", method.Type, err)
			failedChannels = append(failedChannels, string(method.Type))
			continue
		}

		s.logger.Printf("通知方式 %s 发送成功: %s", method.Type, message)
		successChannels = append(successChannels, string(method.Type))
	}

	if len(successChannels) == 0 {
		return mcp.NewToolResultError("所有通知方式均发送失败"), nil
	}

	resultMsg := fmt.Sprintf("通知成功，成功渠道: %s", strings.Join(successChannels, ", "))
	if len(failedChannels) > 0 {
		resultMsg = fmt.Sprintf("%s；失败渠道: %s", resultMsg, strings.Join(failedChannels, ", "))
	}
	return mcp.NewToolResultText(resultMsg), nil
}
