package testdb

import "time"

// RowValues 是 persistence.Row 的通用测试替身：按列顺序提供 Values，
// Scan 按目标类型断言赋值。支持各模块 repo 测试所需的全部通用类型
// （模块域类型如 *UserAccountStatus/*MediaCategory 由模块内自行扩展）。
type RowValues struct {
	Values []any
	Err    error
}

func (r RowValues) Scan(dest ...any) error {
	if r.Err != nil {
		return r.Err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.Values[i].(string)
		case *bool:
			*d = r.Values[i].(bool)
		case *int:
			*d = r.Values[i].(int)
		case *float64:
			*d = r.Values[i].(float64)
		case *time.Time:
			switch v := r.Values[i].(type) {
			case time.Time:
				*d = v
			case string:
				t, _ := time.Parse(time.RFC3339Nano, v)
				*d = t
			}
		case *[]string:
			*d = append([]string(nil), r.Values[i].([]string)...)
		case **string:
			if r.Values[i] == nil {
				*d = nil
			} else {
				switch v := r.Values[i].(type) {
				case string:
					copied := v
					*d = &copied
				case *string:
					*d = v
				}
			}
		case **int:
			if r.Values[i] == nil {
				*d = nil
			} else {
				switch v := r.Values[i].(type) {
				case int:
					copied := v
					*d = &copied
				case *int:
					*d = v
				}
			}
		case **time.Time:
			if r.Values[i] == nil {
				*d = nil
			} else {
				switch v := r.Values[i].(type) {
				case time.Time:
					copied := v
					*d = &copied
				case *time.Time:
					*d = v
				}
			}
		}
	}
	return nil
}

// RowsValues 是 persistence.Rows 的通用测试替身：按行提供 [][]any，
// 可选 TailErr（模拟 rows.Err 非 nil）。
type RowsValues struct {
	Rows    [][]any
	Idx     int
	TailErr error
}

func (r *RowsValues) Next() bool { return r.Idx < len(r.Rows) }
func (r *RowsValues) Scan(dest ...any) error {
	row := RowValues{Values: r.Rows[r.Idx]}
	r.Idx++
	return row.Scan(dest...)
}
func (r *RowsValues) Close() error { return nil }
func (r *RowsValues) Err() error   { return r.TailErr }
