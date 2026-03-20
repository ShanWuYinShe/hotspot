package persistence

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }
