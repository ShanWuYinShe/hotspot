package testdb

import "testing"

// 双实现契约一致性测试基建（审查遗留项：repo_memory / repo_postgres 漂移是本
// 项目最大 bug 温床，四轮共修 35+ 处，但缺「同一组用例跑两实现」的固定范式）。
//
// 用法（见 modules/privatechats/list_contract_test.go 的首个接入示例）：
//
//	testdb.RunContract(t,
//	    []testdb.ContractImpl[Repository, contractState]{
//	        {Name: "memory", New: ...},
//	        {Name: "postgres", New: ...}, // 真实库，建议用 BeginScope 包事务
//	    },
//	    []testdb.ContractCase[Repository, contractState]{...},
//	)
//
// 设计约束：
//   - New 返回「实现 + 用例上下文（已就绪的 id/ctx）」，因为真实库实现需要先
//     插入 users 等前置行，内存实现则不需要——准备逻辑属于实现自身；
//   - 每个用例都拿全新实现实例（隔离），任一实现失败会带上实现名，便于判断是
//     「两实现漂移」还是「共同 bug」；
//   - cleanup 在用例结束后执行（真实库实现用于回滚事务，避免污染测试库）。

// ContractImpl 一个参与契约测试的实现及其准备逻辑。
type ContractImpl[T any, S any] struct {
	Name string
	// New 建立被测实现并返回用例上下文；cleanup 可为 nil。
	New func(t *testing.T) (impl T, state S, cleanup func())
}

// ContractCase 一条契约用例：断言两实现输出一致的行为。
type ContractCase[T any, S any] struct {
	Name string
	Run  func(t *testing.T, impl T, state S)
}

// RunContract 依次在全部实现上运行全部用例。
func RunContract[T any, S any](t *testing.T, impls []ContractImpl[T, S], cases []ContractCase[T, S]) {
	t.Helper()
	if len(impls) == 0 || len(cases) == 0 {
		t.Fatal("contract: impls and cases must not be empty")
	}
	for _, impl := range impls {
		impl := impl
		t.Run(impl.Name, func(t *testing.T) {
			for _, tc := range cases {
				tc := tc
				t.Run(tc.Name, func(t *testing.T) {
					implValue, state, cleanup := impl.New(t)
					if cleanup != nil {
						defer cleanup()
					}
					tc.Run(t, implValue, state)
				})
			}
		})
	}
}
