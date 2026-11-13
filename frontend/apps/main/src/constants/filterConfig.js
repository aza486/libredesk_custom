export const LOGIC = {
    AND: 'AND',
    OR: 'OR'
}

// Mirrors dbutil.MaxFilterGroups on the backend.
export const MAX_FILTER_GROUPS = 10

export const FIELD_TYPE = {
    SELECT: 'select',
    TAG: 'tag',
    MULTI_SELECT: 'multi-select',
    TEXT: 'text',
    NUMBER: 'number',
    RICHTEXT: 'richtext',
    BOOLEAN: 'boolean',
    DATE: 'date',
    WEBHOOK: 'webhook',
    RECIPIENTS: 'recipients',
}

export const OPERATOR = {
    EQUALS: 'equals',
    NOT_EQUALS: 'not equals',
    SET: 'set',
    NOT_SET: 'not set',
    CONTAINS: 'contains',
    NOT_CONTAINS: 'not contains',
    GREATER_THAN: 'greater than',
    LESS_THAN: 'less than',
    BETWEEN: 'between',
    STARTS_WITH: 'starts with'
}

// Labels are presentation-only; operators remain the persisted/API values.
export const OPERATOR_LABEL_KEYS = {
    [OPERATOR.EQUALS]: 'filter.operator.equals',
    [OPERATOR.NOT_EQUALS]: 'filter.operator.notEquals',
    [OPERATOR.SET]: 'filter.operator.set',
    [OPERATOR.NOT_SET]: 'filter.operator.notSet',
    [OPERATOR.CONTAINS]: 'filter.operator.contains',
    [OPERATOR.NOT_CONTAINS]: 'filter.operator.notContains',
    [OPERATOR.GREATER_THAN]: 'filter.operator.greaterThan',
    [OPERATOR.LESS_THAN]: 'filter.operator.lessThan',
    [OPERATOR.BETWEEN]: 'filter.operator.between',
    [OPERATOR.STARTS_WITH]: 'filter.operator.startsWith'
}
export const operatorLabel = (op, t, multiValue = true) => {
    if (multiValue && op === OPERATOR.CONTAINS) return t('filter.containsAnyOf')
    if (multiValue && op === OPERATOR.NOT_CONTAINS) return t('filter.containsNoneOf')
    const key = OPERATOR_LABEL_KEYS[op]
    return key ? t(key) : op
}

export const FIELD_OPERATORS = {
    SELECT: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.SET, OPERATOR.NOT_SET],
    BOOLEAN: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS],
    TEXT: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.CONTAINS,
        OPERATOR.NOT_CONTAINS
    ],
    // "starts with" is only implemented by the automation evaluator, not the SQL filter builder.
    TEXT_AUTOMATION: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.CONTAINS,
        OPERATOR.NOT_CONTAINS,
        OPERATOR.STARTS_WITH
    ],
    // For text columns that do not support partial matching, only allow exact match operators.
    TEXT_EXACT: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.SET, OPERATOR.NOT_SET],
    DATE: [
        OPERATOR.EQUALS,
        OPERATOR.NOT_EQUALS,
        OPERATOR.SET,
        OPERATOR.NOT_SET,
        OPERATOR.GREATER_THAN,
        OPERATOR.LESS_THAN,
        OPERATOR.BETWEEN
    ],
    NUMBER: [OPERATOR.EQUALS, OPERATOR.NOT_EQUALS, OPERATOR.GREATER_THAN, OPERATOR.LESS_THAN],
    MULTI_SELECT: [OPERATOR.CONTAINS, OPERATOR.NOT_CONTAINS, OPERATOR.SET, OPERATOR.NOT_SET]
}
