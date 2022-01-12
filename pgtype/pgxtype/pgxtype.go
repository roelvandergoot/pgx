package pgxtype

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}

// LoadDataType uses conn to inspect the database for typeName and produces a pgtype.DataType suitable for
// registration on ci.
func LoadDataType(ctx context.Context, conn Querier, ci *pgtype.ConnInfo, typeName string) (pgtype.DataType, error) {
	var oid uint32

	err := conn.QueryRow(ctx, "select $1::text::regtype::oid;", typeName).Scan(&oid)
	if err != nil {
		return pgtype.DataType{}, err
	}

	var typtype string

	err = conn.QueryRow(ctx, "select typtype::text from pg_type where oid=$1", oid).Scan(&typtype)
	if err != nil {
		return pgtype.DataType{}, err
	}

	switch typtype {
	case "b": // array
		elementOID, err := GetArrayElementOID(ctx, conn, oid)
		if err != nil {
			return pgtype.DataType{}, err
		}

		var elementCodec pgtype.Codec
		if dt, ok := ci.DataTypeForOID(elementOID); ok {
			if dt.Codec == nil {
				return pgtype.DataType{}, errors.New("array element OID not registered with Codec")
			}
			elementCodec = dt.Codec
		}

		return pgtype.DataType{Name: typeName, OID: oid, Codec: &pgtype.ArrayCodec{ElementOID: elementOID, ElementCodec: elementCodec}}, nil
	case "c": // composite
		panic("TODO - restore composite support")
		// fields, err := GetCompositeFields(ctx, conn, oid)
		// if err != nil {
		// 	return pgtype.DataType{}, err
		// }
		// ct, err := pgtype.NewCompositeType(typeName, fields, ci)
		// if err != nil {
		// 	return pgtype.DataType{}, err
		// }
		// return pgtype.DataType{Value: ct, Name: typeName, OID: oid}, nil
	case "e": // enum
		return pgtype.DataType{Name: typeName, OID: oid, Codec: &pgtype.EnumCodec{}}, nil
	default:
		return pgtype.DataType{}, errors.New("unknown typtype")
	}
}

func GetArrayElementOID(ctx context.Context, conn Querier, oid uint32) (uint32, error) {
	var typelem uint32

	err := conn.QueryRow(ctx, "select typelem from pg_type where oid=$1", oid).Scan(&typelem)
	if err != nil {
		return 0, err
	}

	return typelem, nil
}

// TODO - restore composite support
// GetCompositeFields gets the fields of a composite type.
// func GetCompositeFields(ctx context.Context, conn Querier, oid uint32) ([]pgtype.CompositeTypeField, error) {
// 	var typrelid uint32

// 	err := conn.QueryRow(ctx, "select typrelid from pg_type where oid=$1", oid).Scan(&typrelid)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var fields []pgtype.CompositeTypeField

// 	rows, err := conn.Query(ctx, `select attname, atttypid
// from pg_attribute
// where attrelid=$1
// order by attnum`, typrelid)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for rows.Next() {
// 		var f pgtype.CompositeTypeField
// 		err := rows.Scan(&f.Name, &f.OID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		fields = append(fields, f)
// 	}

// 	if rows.Err() != nil {
// 		return nil, rows.Err()
// 	}

// 	return fields, nil
// }
