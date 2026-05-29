package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"review-view/internal/model"
	"review-view/internal/store"
)

// TaskCache 在内存中缓冲任务日志和流式结果，定期批量写入 DB。
// 日志 flush 后从内存删除（已消费），结果 flush 只写 DB 不删除（SSE 需要完整快照）。
type TaskCache struct {
	mu            sync.Mutex
	logBuffers    map[int64][]model.TaskLog  // taskID -> 待刷盘日志
	resultBuffers map[int64]*strings.Builder // taskID -> 累积结果
	tokenBuffers  map[int64][2]int64         // taskID -> [inputTokens, outputTokens]
	store         store.TaskStore
	notify        chan int64 // 有新数据时发送 taskID
	flushInterval time.Duration
}

func NewTaskCache(store store.TaskStore) *TaskCache {
	return &TaskCache{
		logBuffers:    make(map[int64][]model.TaskLog),
		resultBuffers: make(map[int64]*strings.Builder),
		tokenBuffers:  make(map[int64][2]int64),
		store:         store,
		notify:        make(chan int64, 64),
		flushInterval: 5 * time.Second,
	}
}

// Notify 返回通知 channel，SSE handler 通过它感知新数据。
func (c *TaskCache) Notify() <-chan int64 {
	return c.notify
}

// AppendLog 写入日志到内存 buffer，不直接写 DB。
func (c *TaskCache) AppendLog(taskID int64, level model.TaskLogLevel, message string) {
	log := model.TaskLog{
		TaskID:    taskID,
		Level:     level,
		Message:   message,
		CreatedAt: time.Now(),
	}

	c.mu.Lock()
	c.logBuffers[taskID] = append(c.logBuffers[taskID], log)
	c.mu.Unlock()

	c.sendNotify(taskID)
}

// AppendResultChunk 追加 LLM 文本块到结果缓冲。
func (c *TaskCache) AppendResultChunk(taskID int64, chunk string) {
	c.mu.Lock()
	if _, ok := c.resultBuffers[taskID]; !ok {
		c.resultBuffers[taskID] = &strings.Builder{}
	}
	c.resultBuffers[taskID].WriteString(chunk)
	c.mu.Unlock()

	c.sendNotify(taskID)
}

// GetLogs 返回指定 task 的内存中日志（用于 SSE 推送）。
func (c *TaskCache) GetLogs(taskID int64) []model.TaskLog {
	c.mu.Lock()
	defer c.mu.Unlock()
	logs := c.logBuffers[taskID]
	out := make([]model.TaskLog, len(logs))
	copy(out, logs)
	return out
}

// GetResult 返回指定 task 的当前累积结果快照。
func (c *TaskCache) GetResult(taskID int64) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if b, ok := c.resultBuffers[taskID]; ok {
		return b.String()
	}
	return ""
}

// Flush 将指定 task 的缓冲日志和结果写入 DB。
// 日志 flush 后从内存删除，结果只写 DB 不删除。
func (c *TaskCache) Flush(taskID int64) {
	c.mu.Lock()
	logs := c.logBuffers[taskID]
	delete(c.logBuffers, taskID)
	result := ""
	if b, ok := c.resultBuffers[taskID]; ok {
		result = b.String()
	}
	c.mu.Unlock()

	if len(logs) > 0 {
		_ = c.store.AppendLogs(logs)
	}
	if result != "" {
		_ = c.store.UpdateResult(taskID, result)
	}
}

// FlushAll 将所有缓冲日志和结果写入 DB。
func (c *TaskCache) FlushAll() {
	c.mu.Lock()
	allLogs := c.logBuffers
	c.logBuffers = make(map[int64][]model.TaskLog)
	allResults := c.resultBuffers
	c.mu.Unlock()

	for _, logs := range allLogs {
		if len(logs) > 0 {
			_ = c.store.AppendLogs(logs)
		}
	}
	for taskID, b := range allResults {
		if b.Len() > 0 {
			_ = c.store.UpdateResult(taskID, b.String())
		}
	}
}

// RemoveResult 任务完成后移除结果缓冲，释放内存。
func (c *TaskCache) RemoveResult(taskID int64) {
	c.mu.Lock()
	delete(c.resultBuffers, taskID)
	c.mu.Unlock()
}

// UpdateTokens 更新任务当前 token 消耗（流式过程中可实时调用）。
func (c *TaskCache) UpdateTokens(taskID int64, input, output int64) {
	c.mu.Lock()
	c.tokenBuffers[taskID] = [2]int64{input, output}
	c.mu.Unlock()
}

// SendNotify 显式触发 SSE 通知，用于任务完成后触发 done 检测。
func (c *TaskCache) SendNotify(taskID int64) {
	c.sendNotify(taskID)
}

// GetTokens 返回任务当前累计 token 数量。
func (c *TaskCache) GetTokens(taskID int64) (int64, int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.tokenBuffers[taskID]
	return t[0], t[1]
}

// StartFlushLoop 启动后台定期刷盘 goroutine。
func (c *TaskCache) StartFlushLoop(ctx context.Context) {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.FlushAll()
			return
		case <-ticker.C:
			c.FlushAll()
		}
	}
}

func (c *TaskCache) sendNotify(taskID int64) {
	select {
	case c.notify <- taskID:
	default:
	}
}
