package dynamictools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/utils"
)

// maxHTTPFetchBody limits response size for schema-driven fetches (memory safety).
const maxHTTPFetchBody = 2 << 20 // 2 MiB

// HostFunctions 提供受控的数据访问能力。
type HostFunctions struct {
	httpClient *http.Client
}

// NewHostFunctions 创建 HostFunctions 实例。
func NewHostFunctions() *HostFunctions {
	return &HostFunctions{
		httpClient: newFetchHTTPClient(),
	}
}

// HTTPFetch 发起 HTTP 请求获取数据。
func (hf *HostFunctions) HTTPFetch(ctx context.Context, method, rawURL string, headers map[string]string, body string) (map[string]any, error) {
	safeURL, err := utils.ValidateURLForRequest(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, safeURL.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := hf.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPFetchBody+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(bodyBytes) > maxHTTPFetchBody {
		return nil, fmt.Errorf("response body exceeds limit (%d bytes)", maxHTTPFetchBody)
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return map[string]any{"text": string(bodyBytes)}, nil
	}

	return result, nil
}

// ReadChannel 读取频道消息（预留接口）。
func (hf *HostFunctions) ReadChannel(ctx context.Context, channelID string, limit int) ([]map[string]any, error) {
	// Phase 1: 预留接口，返回空数据
	return nil, nil
}

// QueryData 查询内部数据（预留接口）。
func (hf *HostFunctions) QueryData(ctx context.Context, query string) (map[string]any, error) {
	// Phase 1: 预留接口
	return nil, nil
}

// SendMessage 通过频道发送消息（预留接口）。
func (hf *HostFunctions) SendMessage(ctx context.Context, channelID, message string) error {
	// Phase 1: 预留接口
	return nil
}
