package dbutil

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// PaginationOptions represents the options for paginating a query
type PaginationOptions struct {
	Page     int
	PageSize int
	OrderBy  string
	Order    string
}

const (
	ASC  string = "ASC"
	DESC string = "DESC"
)

// Filter represents a single filter condition
type Filter struct {
	Model    string `json:"model"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// AllowedFields represents the allowed fields for each model
type AllowedFields map[string][]string

// PaginateAndFilterQuery returns a paginated and filtered query with arguments
func PaginateAndFilterQuery(baseQuery string, existingArgs []interface{}, opts PaginationOptions, filtersJSON string, allowedFields AllowedFields) (string, []interface{}, error) {
	// Parse filters
	var filters []Filter
	if filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return "", nil, fmt.Errorf("invalid filters JSON: %v", err)
		}
	}

	// Apply filters
	query, args, err := applyFilters(baseQuery, existingArgs, filters, allowedFields)
	if err != nil {
		return "", nil, err
	}

	// Validate and set default values for pagination
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 10 // Default page size
	}

	// Calculate offset
	offset := (opts.Page - 1) * opts.PageSize

	// Prepare the order by clause
	var orderByClause string
	if opts.OrderBy != "" {
		orderByClause = fmt.Sprintf("ORDER BY %s", opts.OrderBy)
		if opts.Order != "" {
			switch strings.ToUpper(opts.Order) {
			case ASC, DESC:
				orderByClause += " " + strings.ToUpper(opts.Order)
			default:
				return "", nil, fmt.Errorf("invalid order direction: %s", opts.Order)
			}
		}
	}

	// Append pagination to the query
	query = fmt.Sprintf("%s %s LIMIT $%d OFFSET $%d",
		query,
		orderByClause,
		len(args)+1,
		len(args)+2,
	)

	// Append pagination arguments
	args = append(args, opts.PageSize, offset)

	return query, args, nil
}

// applyFilters applies passed filters to the based query
func applyFilters(baseQuery string, existingArgs []interface{}, filters []Filter, allowedFields AllowedFields) (string, []interface{}, error) {
	var conditions []string
	args := make([]interface{}, len(existingArgs))
	copy(args, existingArgs)

	operatorMap := map[string]string{
		"=":  "=",
		"!=": "!=",
		">":  ">",
		"<":  "<",
		">=": ">=",
		"<=": "<=",
	}

	for _, filter := range filters {
		modelFields, ok := allowedFields[filter.Model]
		if !ok {
			return "", nil, fmt.Errorf("invalid model in filter: %s", filter.Model)
		}

		if !slices.Contains(modelFields, filter.Field) {
			return "", nil, fmt.Errorf("invalid field in filter: %s for model: %s", filter.Field, filter.Model)
		}

		op, ok := operatorMap[filter.Operator]
		if !ok {
			return "", nil, fmt.Errorf("invalid operator: %s", filter.Operator)
		}

		condition := fmt.Sprintf("%s.%s %s $%d", filter.Model, filter.Field, op, len(args)+1)
		conditions = append(conditions, condition)
		args = append(args, filter.Value)
	}

	if len(conditions) > 0 {
		if strings.Contains(baseQuery, "WHERE") {
			baseQuery += " AND " + strings.Join(conditions, " AND ")
		} else {
			baseQuery += " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	return baseQuery, args, nil
}
