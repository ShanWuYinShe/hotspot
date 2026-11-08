package comments

// 以下辅助从生产代码下沉到测试：生产路径已不使用，仅测试替身实现需要。

func parentIDStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
