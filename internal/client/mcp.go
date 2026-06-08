package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// --- JSON-RPC 2.0 types ---

// JSONRPCRequest is a JSON-RPC 2.0 request.
// When ID is nil, it's a notification (server won't send a response body).
type JSONRPCRequest struct {
	JSONRPC 	string			`json:"jsonrpc"`
	Method		string			`json:"method"`
	Params		interface{}	`json:"params,omitempty"`
	ID 				*int				`json:"id,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC 	string					`json:"jsonrpc"`
	ID 				*int 						`json:"id,omitempty"`
	Result		json.RawMessage	`json:"result,omitempty"`
	Error 		*JSONRPCError		`json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error object.
type JSONRPCError struct {
	Code			int			`json:"code"`
	Message 	string	`json:"message"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// --- MCP protocol types ---
type initializeParams struct {
	ProtocolVersion string 			`json:"protocolVersion"`
	Capabilities		struct{}		`json:"capabilities"`
	ClientInfo			clientInfo	`json:"clientInfo"`
}

type clientInfo struct {
	Name 			string `json:"name"`
	Version 	string `json:"version"`
}

type toolCallParams struct {
	Name 				string			`json:"name"`
	Arguments		interface{}	`json:"arguments,omitempty"`
}

// toolCallResult is the MCP result shape from tools/call.
type toolCallResult struct {
	Content	 []contentBlock 	`json:"content"`
	IsError		bool						`json:"isError"`
}

type contentBlock struct {
	Type 	string 	`json:"type"`
	Text	string	`json:"text"`
}

// --- Domain types ---

// Shard represents a single memory fragment from search results.
type Shard struct {
	ID				string		`json:"id"`
	Score			float64		`json:"score"`
	Content		string		`json:"content"`
}

// Bond represents a relational connection between two shards.
type Bond struct {
	From			string 		`json:"from"`
	To 				string 		`json:"to"`
	Strength	float64		`json:"strength"`
}

// SearchResult holds parsed output from search_all.
type SearchResult struct {
	Shards	[]Shard		`json:"shards"`
	Bonds		[]Bond		`json:"bonds"`
}

// StatusResponse holds parsed output from get_status.
type StatusResponse struct {
	Mesh			MeshStats			`json:"mesh"`
	Services	ServiceHealth	`json:"services"`
}

type MeshStats struct {
	Shards				int `json:"shards"`
	Bonds					int `json:"bonds"`
	Communities		int `json:"communities"`
}

type ServiceHealth struct {
	Hub				string `json:"hub"`
	Neo4j			string `json:"neo4j"`
	Postgres	string `json:"postgres"`
}

// GetStatus calls the get_status tool and returns mesh health data.
func (c *MCPClient) GetStatus() (*StatusResponse, error) {
	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name: "get_status",
	})
	if err != nil {
		return nil, fmt.Errorf("get_status request failed: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("get_status error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("get_status returned an error")
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("get_status returned empty response")
	}

	// Unlike search_all, the text content IS valid JSON — unmarshal directly
	var status StatusResponse
	if err := json.Unmarshal([]byte(result.Content[0].Text), &status); err != nil {
		return nil, fmt.Errorf("failed to parse status JSON: %w", err)
	}

	return &status, nil
}

// ShardDetail represents a full shard returned by get_shard / get_core_shards.
type ShardDetail struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// SaveInput holds the parameters for saving a memory shard.
type SaveInput struct {
	ID 					string `json:"id"`
	Content 		string `json:"content"`
	Category 		string `json:"category"`
}

// SaveMemory calls the save_memory tool and returns the confirmation message.
func (c *MCPClient) SaveMemory(input SaveInput) (string, error) {
	args := map[string]interface{}{
		"id": 				input.ID,
		"content":		input.Content,
		"category":		input.Category,
	}

	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name:				"save_memory",
		Arguments:	args,
	})
	if err != nil {
		return "", fmt.Errorf("save_memory request failed: %w", err)
	}
	if resp.Error != nil {
		return "", resp.Error
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return "", fmt.Errorf("save_memory error: %s", result.Content[0].Text)
		}
		return "", fmt.Errorf("save_memory returned an error")
	}

	if len(result.Content) == 0 {
		return "saved (no confirmation from server)", nil
	}

	return result.Content[0].Text, nil
}

// GetShardByID calls the get_shard tool and returns a single shard.
func (c *MCPClient) GetShardByID(id string) (*ShardDetail, error) {
	args := map[string]interface{}{
		"id": id,
	}

	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name:      "get_shard",
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("get_shard request failed: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("get_shard error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("get_shard returned an error")
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("get_shard returned empty response")
	}

	var shard ShardDetail
	if err := json.Unmarshal([]byte(result.Content[0].Text), &shard); err != nil {
		return nil, fmt.Errorf("failed to parse shard JSON: %w", err)
	}

	return &shard, nil
}

// GetCoreShards calls the get_core_shards tool and returns all core shards.
func (c *MCPClient) GetCoreShards() ([]ShardDetail, error) {
	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name: "get_core_shards",
	})
	if err != nil {
		return nil, fmt.Errorf("get_core_shards request failed: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("get_core_shards error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("get_core_shards returned an error")
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("get_core_shards returned empty response")
	}

	var shards []ShardDetail
	if err := json.Unmarshal([]byte(result.Content[0].Text), &shards); err != nil {
		return nil, fmt.Errorf("failed to parse core shards JSON: %w", err)
	}

	return shards, nil
}

// --- The client ---

// MCPClient holds connection state for a single MCP session.
type MCPClient struct {
	endpoint 		string
	apiKey			string
	sessionID		string
	nextID			int
	httpClient	*http.Client
}

// NewMCPClient creates a client and performs the MCP initialization handshake.
// After this returns, the session is live and ready for tools/call requests.
func NewMCPClient(endpoint, apiKey string) (*MCPClient, error) {
	c := &MCPClient{
		endpoint: endpoint,
		apiKey: 	apiKey,
		nextID: 	0,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Step 1: initialize — exchange capabilities with the server
	initParam := initializeParams{
		ProtocolVersion: "2025-03-26",
		ClientInfo: clientInfo{
			Name:			"shard-cli",
			Version: 	"0.1.0",
		},
	}

	resp, err := c.sendRequest("initialize", initParam)
	if err != nil {
		return nil, fmt.Errorf("MCP initialize failed: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Step 2: notifications/initialized — tell server we're ready
	if err := c.sendNotification("notifications/initialized"); err != nil {
		return nil, fmt.Errorf("MCP initialized notification failed: %w", err)
	}

	return c, nil
}

// SearchAll calls the search_all tool and returns parsed results.
func (c *MCPClient) SearchAll(query string, limit int, bias float64) (*SearchResult, error) {
	args := map[string]interface{}{
		"query": query,
		"limit": limit,
		"bias": bias,
	}

	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name:				"search_all",
		Arguments:	args,
	})
	if err != nil {
		return nil, fmt.Errorf("search_all request failed: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Unmarshal the MCP tool result envelope
	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("search_all error: %s",
		result.Content[0].Text)
		}
		return nil, fmt.Errorf("search_all returned an error")
	}
	// The tool returns a single text block — parse it into structured data
	if len(result.Content) == 0 {
		return &SearchResult{}, nil
	}

	return parseSearchResult(result.Content[0].Text), nil
}

// --- HTTP transport layer ---

// sendRequest sends a JSON-RPC request (with an ID) and returns the response.
// Handles both application/json and text/event-stream responses.
func (c *MCPClient) sendRequest(method string, params interface{}) (*JSONRPCResponse, error) {
	c.nextID++
	id := c.nextID

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method: method,
		Params: params,
		ID:			&id,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// Store session ID if server provides one
	if sid := httpResp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID = sid
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Branch based on content type
	contentType := httpResp.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "text/event-stream") {
		return c.readSSEResponse(httpResp.Body, id)
	}

	// Default: application/json
	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return &rpcResp, nil
}

// sendNotification sends a JSON-RPC notification (no ID, no response expected).
func (c *MCPClient) sendNotification(method string) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method: method,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if sid := httpResp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID = sid
	}

	// Notifications expect 2xx (typically 202 Accepted)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// readSSEResponse reads a Server-Sent Events stream and extracts the JSON-RPC
// response matching the given request ID.
func (c *MCPClient) readSSEResponse(body io.Reader, id int) (*JSONRPCResponse, error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE data lines start with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var rpcResp JSONRPCResponse
		if err := json.Unmarshal([]byte(data), &rpcResp); err != nil {
			continue // skip malformed lines
		}

		// Match by request ID
		if rpcResp.ID != nil && *rpcResp.ID == id {
			return &rpcResp, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("SSE stream error: %w", err)
	}

	return nil, fmt.Errorf("no matching response found in SSE stream for id %d", id)
}

// --- Response text parser ---

// Regex patterns for parsing the search_all text reponse.
// Shard format: [shard-id] (Score: 0.42): Content text here...
// Bond format: - shard-a <-> shard-b (Strength: 0.72)
var (
	shardRegex = regexp.MustCompile(`\[(.+?)\] \(Score: ([\d.]+)\): (.+)`)
	bondRegex  = regexp.MustCompile(`- (.+?) <-> (.+?) \(Strength: ([\d.]+)\)`)
)

// parseSearchResult extracts structured data from the search_all text response.
func parseSearchResult(text string) *SearchResult {
	result := &SearchResult{}

	// Split into shard section and bonds section
	parts := strings.SplitN(text, "Relational Bonds", 2)

	// Parse shards from the first section
	// Split by "---" delimiter — each block between delimiters is one shard
	shardBlocks := strings.Split(parts[0], "---")
	for _, block := range shardBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		matches := shardRegex.FindStringSubmatch(block)
		if matches == nil {
			continue
		}

		score, _ := strconv.ParseFloat(matches[2], 64)

		result.Shards = append(result.Shards, Shard{
			ID:				matches[1],
			Score: 		score,
			Content: 	matches[3],
		})
	}

	// Parse bonds from the second section (if present)
	if len(parts) > 1 {
		bondMatches := bondRegex.FindAllStringSubmatch(parts[1], -1)
		for _, m := range bondMatches {
			strength, _ := strconv.ParseFloat(m[3], 64)
			result.Bonds = append(result.Bonds, Bond{
				From:				m[1],
				To: 				m[2],
				Strength: 	strength,
			})
		}
	}

	return result
}