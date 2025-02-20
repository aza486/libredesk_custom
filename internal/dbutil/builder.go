package dbutil

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// PaginationOptions represents the options for paginating a query.
type PaginationOptions struct {
	Page     int
	PageSize int
	OrderBy  string
	Order    string
}

// Order directions.
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// Filter represents a filter to be applied to a query.
type Filter struct {
	Model    string `json:"model"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// AllowedFields is a map of model names to a list of allowed fields for that model.
type AllowedFields map[string][]string

// BuildPaginatedQuery builds a paginated query from the given base query, existing arguments, pagination options, filters JSON, and allowed fields.
func BuildPaginatedQuery(baseQuery string, existingArgs []interface{}, opts PaginationOptions, filtersJSON string, allowedFields AllowedFields) (string, []interface{}, error) {
	if opts.Page <= 0 {
		return "", nil, fmt.Errorf("invalid page number: %d", opts.Page)
	}
	if opts.PageSize <= 0 {
		return "", nil, fmt.Errorf("invalid page size: %d", opts.PageSize)
	}

	var filters []Filter
	if filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return "", nil, fmt.Errorf("invalid filters JSON: %w", err)
		}
	}

	whereClause, filterArgs, err := buildWhereClause(filters, existingArgs, allowedFields)
	if err != nil {
		return "", nil, err
	}

	query := baseQuery
	args := existingArgs

	if whereClause != "" {
		query += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	if opts.OrderBy != "" {
		order := strings.ToUpper(opts.Order)
		if order != "" && order != ASC && order != DESC {
			return "", nil, fmt.Errorf("invalid order direction: %s", opts.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s %s NULLS LAST", opts.OrderBy, order)
	}

	offset := (opts.Page - 1) * opts.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, opts.PageSize, offset)

	return query, args, nil
}

// buildWhereClause builds a WHERE clause from the given filters and returns the WHERE clause and the arguments to be passed to the query.
func buildWhereClause(filters []Filter, existingArgs []interface{}, allowedFields AllowedFields) (string, []interface{}, error) {
	conditions := []string{}
	args := []interface{}{}
	paramCount := len(existingArgs) + 1

	for _, f := range filters {
		modelFields, ok := allowedFields[f.Model]
		if !ok {
			return "", nil, fmt.Errorf("invalid model: %s", f.Model)
		}
		if !slices.Contains(modelFields, f.Field) {
			return "", nil, fmt.Errorf("invalid field: %s for model: %s", f.Field, f.Model)
		}

		field := fmt.Sprintf("%s.%s", f.Model, f.Field)

		switch f.Operator {
		case "equals":
			conditions = append(conditions, field+fmt.Sprintf(" = $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "not equals":
			conditions = append(conditions, field+fmt.Sprintf(" != $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "set":
			conditions = append(conditions, field+" IS NOT NULL")
		case "not set":
			conditions = append(conditions, field+" IS NULL")
		case "in":
			var arr []string
			if err := json.Unmarshal([]byte(f.Value), &arr); err != nil {
				return "", nil, fmt.Errorf("invalid array format for 'in' operator: %v", err)
			}
			placeholders := make([]string, len(arr))
			for i, v := range arr {
				placeholders[i] = fmt.Sprintf("$%d", paramCount)
				args = append(args, v)
				paramCount++
			}
			conditions = append(conditions, field+" IN ("+strings.Join(placeholders, ",")+")")
		case "between":
			values := strings.Split(f.Value, ",")
			if len(values) != 2 {
				return "", nil, fmt.Errorf("between requires 2 values")
			}
			conditions = append(conditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", field, paramCount, paramCount+1))
			args = append(args, strings.TrimSpace(values[0]), strings.TrimSpace(values[1]))
			paramCount += 2
		default:
			return "", nil, fmt.Errorf("invalid operator: %s", f.Operator)
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}
