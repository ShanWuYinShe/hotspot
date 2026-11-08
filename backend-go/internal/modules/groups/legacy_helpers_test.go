package groups

// 以下辅助从生产代码下沉到测试：生产路径已不使用，仅测试替身实现需要。

func groupMutedUntil(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}
