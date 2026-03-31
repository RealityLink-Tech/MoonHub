// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Message Tool Plugin
//
// Copyright (c) 2026 MoonHub contributors

package message

import (
	"context"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

func init() {
	plugin.RegisterPlugin(&MessageToolPlugin{})
}

type MessageToolPlugin struct {
	ctx *plugin.RuntimeContext
}

func (p *MessageToolPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-tool-message",
		Name:        "Message",
		Type:        plugin.TypeTool,
		Version:     "1.0.0",
		Description: "Send messages to chat channels",
		Priority:    100,
	}
}

func (p *MessageToolPlugin) Init(ctx *plugin.RuntimeContext) error {
	p.ctx = ctx
	return nil
}

func (p *MessageToolPlugin) Validate(cfg *config.Config) error {
	return nil
}

func (p *MessageToolPlugin) CreateTools(ctx *plugin.RuntimeContext) []tools.Tool {
	if !ctx.Config.Tools.IsToolEnabled("message") {
		return nil
	}
	msgBus := ctx.Bus
	if msgBus == nil {
		return nil
	}
	messageTool := tools.NewMessageTool()
	messageTool.SetSendCallback(func(channel, chatID, content string) error {
		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		return msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: content,
		})
	})
	return []tools.Tool{messageTool}
}

func (p *MessageToolPlugin) IsCore() bool {
	return true
}
