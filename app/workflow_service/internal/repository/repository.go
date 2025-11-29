package repository

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
)

func DbExecAndReturnCount(execer sqlx.ExtContext, sqlizer sq.Sqlizer) (int, error) {
	var dest int
	query, args, err := sqlizer.ToSql()
	logging.Sugar.Debugw("statement", "sql", query, "args", args)
	if err != nil {
		logging.Sugar.Error(err)
		return 0, err
	}
	query = execer.Rebind(query)
	logging.Sugar.Debug(query, args)
	sqlErr := sqlx.GetContext(context.Background(), execer, &dest, query, args...)
	if sqlErr != nil {
		logging.Sugar.Warn(sqlErr)
		if sqlErr == sql.ErrNoRows {
			return 0, nil
		}
		return 0, sqlErr
	}
	return dest, nil
}

func DbExecAndReturnOne[T any](execer sqlx.ExtContext, sqlizer sq.Sqlizer) (*T, error) {
	var dest T
	query, args, err := sqlizer.ToSql()
	logging.Sugar.Debugw("statement", "sql", query, "args", args)
	if err != nil {
		logging.Sugar.Error(err)
		return nil, err
	}
	query = execer.Rebind(query)
	logging.Sugar.Debug(query, args)
	sqlErr := sqlx.GetContext(context.Background(), execer, &dest, query, args...)
	if sqlErr != nil {
		logging.Sugar.Warn(sqlErr)
		if sqlErr == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sqlErr
	}
	return &dest, nil
}

func DbExecAndReturnOneOld[T any](execer sqlx.ExtContext, query string, args ...interface{}) (*T, error) {
	var dest T
	query = execer.Rebind(query)
	logging.Sugar.Debug(query, args)
	err := sqlx.GetContext(context.Background(), execer, &dest, query, args...)
	if err != nil {
		logging.Sugar.Warn(err)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &dest, nil
}

// func DbExecAndReturnMany[T any](DB *sqlx.DB, query string, args ...interface{}) ([]T, error) {
// 	var dest []T
// 	logging.Sugar.Debug(query, args)
// 	err := DB.Select(&dest, query, args...)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return []T{}, nil
// 		}
// 		return []T{}, err
// 	}
// 	return dest, nil
// }

func DbExecAndReturnMany[T any](execer sqlx.ExtContext, sqlizer sq.Sqlizer) ([]T, error) {
	var dest []T
	query, args, err := sqlizer.ToSql()
	logging.Sugar.Debugw("statement", "sql", query, "args", args)
	if err != nil {
		logging.Sugar.Error(err)
		return nil, err
	}
	query = execer.Rebind(query)
	sqlErr := sqlx.SelectContext(context.Background(), execer, &dest, query, args...)
	if sqlErr != nil {
		logging.Sugar.Errorw("Error in query", "query", query, "args", args, "err", sqlErr)
		if sqlErr == sql.ErrNoRows {
			return []T{}, nil
		}
		return []T{}, sqlErr
	}

	if dest == nil {
		return []T{}, nil
	}
	return dest, nil
}

func DbExecAndReturnManyOld[T any](execer sqlx.ExtContext, query string, args ...interface{}) ([]T, error) {
	var dest []T
	query = execer.Rebind(query)
	logging.Sugar.Debugw("Executing query", "query", query, "args", args)
	err := sqlx.SelectContext(context.Background(), execer, &dest, query, args...)
	if err != nil {
		logging.Sugar.Errorw("Error in query", "query", query, "args", args, "err", err)
		if err == sql.ErrNoRows {
			return []T{}, nil
		}
		return []T{}, err
	}

	if dest == nil {
		return []T{}, nil
	}
	return dest, nil
}

func DbSelectOne[T any](DB *sqlx.DB, query string, args ...interface{}) (*T, error) {
	var dest T
	logging.Sugar.Debug(query, args)
	err := DB.Get(&dest, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &dest, nil
}

// transaction function
func Transact[T any](DB *sqlx.DB, fn func(*sqlx.Tx) (*T, error)) (*T, error) {
	tx, err := DB.Beginx()
	if err != nil {
		return nil, err
	}

	result, err := fn(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	commitErr := tx.Commit()

	if commitErr != nil {
		return nil, commitErr
	}

	return result, nil
}

func GenerateKeyValueQuery(payload map[string]types.Nullable[any]) map[string]interface{} {
	objects := make(map[string]interface{})
	logging.Sugar.Debugf("payload: %v", payload)

	for key, val := range payload {
		logging.Sugar.Debugf("key: %v", key)
		logging.Sugar.Debugf("set: %v", val.Set)
		if val.Set {
			objects[key] = val.Value
		}
	}

	logging.Sugar.Debugf("objects: %v", objects)

	return objects
}
