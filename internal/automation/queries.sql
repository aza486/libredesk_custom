-- name: get-rules
select er.id, er.type, ec.field, ec."operator", ec.value, ec.group_id, ecg.logical_op from engine_rules er inner join engine_conditions ec on ec.rule_id = er.id  
inner join engine_condition_groups ecg on ecg.id = ec.group_id;

-- name: get-rule-actions
select rule_id, action_type, action from engine_actions;