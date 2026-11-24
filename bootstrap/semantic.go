// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main
import (
	"strconv"; "strings"
)
type constantValue struct {
	kind          string; text          string
	integer       int64; resultPayload string
	array         []constantValue
}
func walkValue(input value, visit func(value) bool) {
	if input.kind == "" || !visit(input) {
		return
	}
	for _, argument := range input.args {
		walkValue(argument, visit)
	}
	for _, item := range input.fields {
		walkValue(item.value, visit)
	}
	if input.left != nil {
		walkValue(*input.left, visit)
	}
	if input.right != nil {
		walkValue(*input.right, visit)
	}
}
func walkStatements(items []statement, visit func(statement) bool) {
	for _, item := range items {
		if !visit(item) {
			continue
		}
		walkStatements(item.body, visit); walkStatements(item.otherwise, visit)
	}
}
func walkProgramValues(program *program, visit func(value) bool) {
	walkValue(program.root, visit)
	for _, item := range program.bindings {
		walkValue(item.value, visit)
	}
	walkStatements(program.statements, func(item statement) bool {
		walkValue(item.value, visit); walkValue(item.condition, visit)
		return true
	})
	for _, function := range program.functions {
		walkValue(function.body, visit)
		walkStatements(function.statements, func(item statement) bool {
			walkValue(item.value, visit); walkValue(item.condition, visit)
			return true
		})
	}
}
func seedRuntimeResultPayloads(program *program) map[string]constantValue {
	payloads := map[string]constantValue{}
	for _, function := range program.functions {
		for _, item := range function.statements {
			if item.kind != "变量" {
				continue
			}
			kind, problem := constantType(item.value, payloads)
			if problem != nil || kind != "运行时结果" {
				continue
			}
			payloads[item.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(item.value, payloads)}
		}
	}
	for _, item := range program.statements {
		if item.kind != "变量" {
			continue
		}
		kind, problem := constantType(item.value, payloads)
		if problem != nil || kind != "运行时结果" {
			continue
		}
		payloads[item.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(item.value, payloads)}
	}
	return payloads
}
func programRootEffects(root value) map[string]bool {
	effects := map[string]bool{}
	for _, field := range root.fields {
		if field.name.text != "能力" || field.value.kind != "调用" || field.value.text != "array" {
			continue
		}
		for _, item := range field.value.args {
			if item.kind == "字符串" {
				effects[item.text] = true
			}
		}
	}
	return effects
}
func expandProgramFunctions(program *program, payloads map[string]constantValue) *issue {
	if len(program.functions) == 0 {
		return nil
	}
	rootEffects := programRootEffects(program.root); functions := make(map[string]functionDecl, len(program.functions))
	for index := range program.functions {
		function := program.functions[index]; effects, problem := functionEffects(function)
		if problem != nil {
			return problem
		}
		function.effects = effects; program.functions[index] = function
		if _, exists := functions[function.name.text]; exists {
			return &issue{code: "S-3060", expected: "唯一函数名", actual: function.name.text, suggestion: "每个函数只声明一次", pos: function.name.pos}
		}
		functions[function.name.text] = function
	}
	for index := range program.functions {
		function := program.functions[index]
		if len(function.statements) == 0 || (len(function.effects) == 0 && (containsStatementKind(function.statements, "凡") || blockFunctionUsesRuntimeTextParameter(function) || blockFunctionObservesResultState(function) || blockFunctionReturnsRuntimeCall(function, functions))) || (len(function.effects) != 0 && (containsStatementKind(function.statements, "凡") || blockFunctionConstructsResultLocal(function))) {
			continue
		}
		environment := make(map[string]value, len(function.parameters))
		for _, parameter := range function.parameters {
			environment[parameter.text] = value{kind: "引用", text: parameter.text, pos: parameter.pos}
		}
		locals := map[string]bool{}; collectFunctionLocals(function.statements, locals)
		body, problem := lowerFunctionStatements(function.statements, environment, locals, functions, 0)
		if problem != nil {
			if problem.pos.line == 0 {
				problem.pos = function.name.pos
			}
			return problem
		}
		for _, item := range function.statements {
			if item.kind == "变量" && isTextReturnCall(item.value) {
				return functionBlockIssue(item.value.pos, "已复制的局部文本", "未复制的文本返回值", "先声明文本局部变量并复制返回值，再在局部作用域使用")
			}
		}
		function.body = body; function.statements = nil
		program.functions[index] = function; functions[function.name.text] = function
	}
	runtimeFunctions, runtimeFunctionNames := recursiveRuntimeFunctions(program.functions, functions); annotateRuntimeFunctionResults(runtimeFunctions)
	annotateRuntimeResultPayloads(program, payloads)
	for _, function := range runtimeFunctions {
		if function.runtimeReturnsResult {
			functions[function.name.text] = function
		}
	}
	for index := range program.functions {
		name := program.functions[index].name.text
		if function, exists := runtimeFunctions[name]; exists {
			program.functions[index].runtimeReturnsResult = function.runtimeReturnsResult; program.functions[index].runtimeResultPayload = function.runtimeResultPayload
		}
	}
	program.runtimeFunctions = runtimeFunctions; program.runtimeFunctionNames = runtimeFunctionNames
	for index := range program.functions {
		function := program.functions[index]
		if function.body.kind == "" {
			remaining := 65536
			for statementIndex := range function.statements {
				statement, problem := expandFunctionStatement(function.statements[statementIndex], functions, runtimeFunctions, 0, &remaining, rootEffects, payloads, function.statements)
				if problem != nil {
					return problem
				}
				function.statements[statementIndex] = statement
			}
		} else {
			remaining := 65536; body, problem := expandFunctionValue(function.body, functions, runtimeFunctions, []string{function.name.text}, 0, &remaining, rootEffects, payloads, function.statements)
			if problem != nil {
				return problem
			}
			function.body = body
		}
		program.functions[index] = function; functions[function.name.text] = function
		if _, runtimeFunction := runtimeFunctions[function.name.text]; runtimeFunction {
			runtimeFunctions[function.name.text] = function
		}
	}
	remaining := 65536; var problem *issue
	for index := range program.bindings {
		program.bindings[index].value, problem = expandFunctionValue(program.bindings[index].value, functions, runtimeFunctions, nil, 0, &remaining, rootEffects, payloads, program.statements)
		if problem != nil {
			return problem
		}
	}
	for index := range program.root.fields {
		program.root.fields[index].value, problem = expandFunctionValue(program.root.fields[index].value, functions, runtimeFunctions, nil, 0, &remaining, rootEffects, payloads, program.statements)
		if problem != nil {
			return problem
		}
	}
	for index := range program.statements {
		program.statements[index], problem = expandFunctionStatement(program.statements[index], functions, runtimeFunctions, 0, &remaining, rootEffects, payloads, program.statements)
		if problem != nil {
			return problem
		}
	}
	return nil
}
func recursiveRuntimeFunctions(ordered []functionDecl, functions map[string]functionDecl) (map[string]functionDecl, []string) {
	graph := make(map[string]map[string]bool, len(ordered))
	for _, function := range ordered {
		calls := map[string]bool{}; collectFunctionCalls(function.body, functions, calls)
		var visitStatements func([]statement)
		visitStatements = func(items []statement) {
			for _, item := range items {
				collectFunctionCalls(item.value, functions, calls); collectFunctionCalls(item.condition, functions, calls)
				visitStatements(item.body); visitStatements(item.otherwise)
			}
		}
		visitStatements(function.statements); graph[function.name.text] = calls
	}
	runtime := map[string]functionDecl{}; names := []string{}
	for _, function := range ordered {
		if len(function.effects) != 0 && function.body.kind != "" {
			continue
		}
		if function.body.kind != "" && !callGraphReaches(graph, function.name.text, function.name.text, map[string]bool{}) {
			continue
		}
		runtime[function.name.text] = function; names = append(names, function.name.text)
	}
	return runtime, names
}
func containsStatementKind(statements []statement, kind string) bool {
	for _, item := range statements {
		if item.kind == kind || containsStatementKind(item.body, kind) || containsStatementKind(item.otherwise, kind) {
			return true
		}
	}
	return false
}
func blockFunctionUsesRuntimeTextParameter(function functionDecl) bool {
	parameters := make(map[string]bool, len(function.parameters))
	for _, parameter := range function.parameters {
		parameters[parameter.text] = true
	}
	textParameters := map[string]bool{}; returnTextParameters := map[string]bool{}
	integerParameters := map[string]bool{}; collectRuntimeTextParameterUses(function.statements, parameters, textParameters)
	collectRuntimeTextReturnParameters(function.statements, parameters, returnTextParameters); collectRuntimeIntegerParameterUses(function.statements, parameters, integerParameters)
	collectRuntimeIntegerParameterUsesValue(function.body, parameters, integerParameters)
	if len(textParameters) != 0 {
		return true
	}
	for parameter := range returnTextParameters {
		if !integerParameters[parameter] {
			return true
		}
	}
	return false
}
func blockFunctionObservesResultState(function functionDecl) bool {
	observed := func(input value) bool {
		found := false
		walkValue(input, func(current value) bool {
			if current.kind == "调用" && (current.text == "是否失败" || current.text == "错误码" || current.text == "结果值") && len(current.args) == 1 {
				found = true
				return false
			}
			return !found
		})
		return found
	}
	if observed(function.body) {
		return true
	}
	found := false
	walkStatements(function.statements, func(item statement) bool {
		found = found || observed(item.value) || observed(item.condition)
		return !found
	})
	return found
}
func blockFunctionConstructsResultLocal(function functionDecl) bool {
	constructs := func(input value) bool {
		found := false
		walkValue(input, func(current value) bool {
			if current.kind == "调用" && (current.text == "成功" || current.text == "失败") {
				found = true
				return false
			}
			return !found
		})
		return found
	}
	found := false
	walkStatements(function.statements, func(item statement) bool {
		found = found || item.kind == "变量" && constructs(item.value)
		return !found
	})
	return found
}
func blockFunctionReturnsRuntimeCall(function functionDecl, functions map[string]functionDecl) bool {
	found := false
	walkStatements(function.statements, func(item statement) bool {
		if item.kind != "返回" {
			return true
		}
		walkValue(item.value, func(current value) bool {
			if current.kind == "调用" {
				if _, exists := functions[current.text]; exists {
					found = true
					return false
				}
			}
			return !found
		})
		return !found
	})
	return found
}
func runtimeResultStateUsesValue(input value, name string, found *bool) bool {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && (current.text == "是否失败" || current.text == "错误码" || current.text == "结果值") && len(current.args) == 1 && current.args[0].kind == "引用" && current.args[0].text == name {
			*found = true
			return false
		}
		return !*found
	})
	return *found
}
func functionResultBranchUsesParameter(input value, name string) bool {
	found := false
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && current.text == "ifElse" && len(current.args) == 3 {
			for _, branch := range current.args[1:] {
				if branch.kind == "引用" && branch.text == name {
					found = true
					return false
				}
			}
		}
		return !found
	})
	return found
}
func functionResultBranchUsesValue(items []statement, name string) bool {
	for _, item := range items {
		if functionResultBranchUsesParameter(item.value, name) || functionResultBranchUsesParameter(item.condition, name) {
			return true
		}
		if functionResultBranchUsesValue(item.body, name) || functionResultBranchUsesValue(item.otherwise, name) {
			return true
		}
	}
	return false
}
func collectBlockCalls(items []statement, calls map[string]bool) {
	for _, item := range items {
		if item.value.kind == "调用" {
			calls[item.value.text] = true
		}
		if item.condition.kind == "调用" {
			calls[item.condition.text] = true
		}
		collectBlockCalls(item.body, calls); collectBlockCalls(item.otherwise, calls)
	}
}
func inferRuntimeResultStateParameters(program *program, function functionDecl, parameters map[string]bool) map[string]bool {
	resultParameters := map[string]bool{}; environment := copyConstantEnvironment(program.constants)
	for _, local := range functionLocalNames(function.statements) {
		if constant, exists := program.constants[local]; exists {
			environment[local] = constant
		}
	}
	for _, item := range program.statements {
		if item.kind != "变量" {
			continue
		}
		kind, problem := constantType(item.value, environment)
		if problem != nil {
			continue
		}
		payload := ""
		if kind == "运行时结果" {
			payload = runtimeResultPayload(item.value, environment)
		}
		environment[item.name.text] = constantValue{kind: kind, resultPayload: payload}
	}
	for _, candidate := range program.functions {
		for index, parameter := range candidate.parameters {
			if _, exists := environment[parameter.text]; exists {
				continue
			}
			kind := runtimeFunctionParameterTypes(candidate)[index]
			if kind == "" {
				kind = "运行时整数"
			}
			payload := ""
			if kind == "运行时结果" {
				payload = runtimeResultParameterPayload(candidate, parameter.text)
			}
			environment[parameter.text] = constantValue{kind: kind, resultPayload: payload}
		}
	}
	var inspectCalls func(value)
	inspectCalls = func(input value) {
		if input.kind == "调用" && input.text == function.name.text {
			for index, argument := range input.args {
				if index >= len(function.parameters) {
					continue
				}
				if argument.kind != "引用" && !isRuntimeResultValue(argument) {
					continue
				}
				kind, problem := constantType(argument, environment)
				if problem == nil && kind == "运行时结果" {
					resultParameters[function.parameters[index].text] = true
				}
			}
		}
		for _, argument := range input.args {
			inspectCalls(argument)
		}
		for _, field := range input.fields {
			inspectCalls(field.value)
		}
		if input.left != nil {
			inspectCalls(*input.left)
		}
		if input.right != nil {
			inspectCalls(*input.right)
		}
	}
	inspectCalls(program.root)
	for _, item := range program.bindings {
		inspectCalls(item.value)
	}
	for _, item := range program.statements {
		inspectCalls(item.value); inspectCalls(item.condition)
	}
	for _, item := range program.functions {
		inspectCalls(item.body)
		for _, statement := range item.statements {
			inspectCalls(statement.value); inspectCalls(statement.condition)
		}
	}
	for name := range parameters {
		resultObserved := false; runtimeResultStateUsesValue(function.body, name, &resultObserved)
		for _, item := range function.statements {
			if runtimeResultStateUsesValue(item.value, name, &resultObserved) || runtimeResultStateUsesValue(item.condition, name, &resultObserved) {
				break
			}
		}
		if resultObserved {
			resultParameters[name] = true
		}
	}
	return resultParameters
}
func inferRuntimeResultParameterPayloads(program *program, functionName string, parameters []token, types []string) (map[string]string, *issue) {
	payloads := map[string]string{}; environment := copyConstantEnvironment(program.constants)
	for _, item := range program.statements {
		if item.kind != "变量" {
			continue
		}
		kind, problem := constantType(item.value, environment)
		if problem != nil {
			continue
		}
		payload := ""
		if kind == "运行时结果" {
			payload = runtimeResultPayload(item.value, environment)
		}
		environment[item.name.text] = constantValue{kind: kind, resultPayload: payload}
	}
	for _, candidate := range program.functions {
		candidateTypes := runtimeFunctionParameterTypes(candidate)
		for index, parameter := range candidate.parameters {
			if _, exists := environment[parameter.text]; exists {
				continue
			}
			kind := candidateTypes[index]
			if kind == "" {
				kind = "运行时整数"
			}
			payload := ""
			if kind == "运行时结果" {
				payload = runtimeResultParameterPayload(candidate, parameter.text)
			}
			environment[parameter.text] = constantValue{kind: kind, resultPayload: payload}
		}
	}
	var problem *issue
	walkProgramValues(program, func(input value) bool {
		if problem != nil {
			return false
		}
		if input.kind == "调用" && input.text == functionName {
			for index, argument := range input.args {
				if index >= len(types) || index >= len(parameters) || types[index] != "运行时结果" {
					continue
				}
				payload := runtimeResultPayload(argument, environment)
				if payload == "" {
					continue
				}
				name := parameters[index].text
				if previous, exists := payloads[name]; exists && isTextResultPayload(previous) != isTextResultPayload(payload) {
					problem = &issue{code: "S-3062", expected: "同一结果形参的载荷形态一致", actual: previous + " 与 " + payload, suggestion: "用不同函数或先转换结果载荷", pos: argument.pos}
					return false
				}
				if isTextResultPayload(payload) {
					payloads[name] = "运行时文本"
				} else {
					payloads[name] = "整数"
				}
			}
		}
		return true
	})
	if problem != nil {
		return nil, problem
	}
	return payloads, nil
}
func inspectResultReturn(input value, environment map[string]constantValue, resultParameters map[string]bool) {
	if input.kind != "调用" {
		return
	}
	if input.text == "ifElse" && len(input.args) == 3 {
		inspectResultReturn(input.args[1], environment, resultParameters); inspectResultReturn(input.args[2], environment, resultParameters)
		return
	}
	if input.kind == "调用" && input.runtimeFunction && input.runtimeFunctionResult {
		return
	}
	if input.text != "成功" && input.text != "失败" && !isFunctionEffect(input.text) {
		return
	}
	for _, argument := range input.args {
		for name, constant := range environment {
			if constant.kind == "运行时结果" && argument.kind == "引用" && argument.text == name {
				resultParameters[name] = true
			}
		}
	}
}
func classifyRuntimeFunctionParameterTypes(program *program) *issue {
	for _, name := range program.runtimeFunctionNames {
		function := program.runtimeFunctions[name]; parameters := make(map[string]bool, len(function.parameters))
		for _, parameter := range function.parameters {
			parameters[parameter.text] = true
		}
		textParameters := map[string]bool{}
		if function.body.kind == "" {
			collectRuntimeTextParameterUses(function.statements, parameters, textParameters)
		} else {
			collectRuntimeTextParameterUsesValue(function.body, parameters, textParameters, nil)
		}
		resultParameters := inferRuntimeResultStateParameters(program, function, parameters)
		for parameter := range resultParameters {
			delete(textParameters, parameter)
		}
		explicitTextParameters := make(map[string]bool, len(textParameters))
		for parameter := range textParameters {
			explicitTextParameters[parameter] = true
		}
		returnTextParameters := map[string]bool{}
		if function.body.kind == "" {
			collectRuntimeTextReturnParameters(function.statements, parameters, returnTextParameters)
		}
		for parameter := range resultParameters {
			delete(returnTextParameters, parameter)
		}
		integerParameters := map[string]bool{}; blockCalls := map[string]bool{}
		collectBlockCalls(function.statements, blockCalls)
		for _, parameter := range function.parameters {
			if blockCalls[parameter.text] && !functionResultBranchUsesValue(function.statements, parameter.text) {
				integerParameters[parameter.text] = true
			}
		}
		collectRuntimeIntegerParameterUses(function.statements, parameters, integerParameters); collectRuntimeIntegerParameterUsesValue(function.body, parameters, integerParameters)
		inferRuntimeTextParametersFromCalls(program, name, function.parameters, textParameters, integerParameters); types := make([]string, len(function.parameters))
		for index, parameter := range function.parameters {
			types[index] = "运行时整数"
			if resultParameters[parameter.text] {
				types[index] = "运行时结果"
				continue
			}
			if explicitTextParameters[parameter.text] ||
				(textParameters[parameter.text] || returnTextParameters[parameter.text]) && !integerParameters[parameter.text] {
				types[index] = "运行时字符串"
			}
		}
		if function.body.kind != "" {
			for _, parameterType := range types {
				if parameterType == "运行时字符串" {
					return &issue{code: "S-3062", expected: "表达式运行时函数的整数参数", actual: "文本参数", suggestion: "先在块体函数中使用文本参数长度桥；表达式运行时文本参数留待后续 ABI", pos: function.name.pos}
				}
			}
		}
		payloads, problem := inferRuntimeResultParameterPayloads(program, name, function.parameters, types)
		if problem != nil {
			return problem
		}
		function.parameterResultPayloads = payloads; function.parameterTypes = types
		program.runtimeFunctions[name] = function
		for index := range program.functions {
			if program.functions[index].name.text == name {
				program.functions[index].parameterTypes = types; program.functions[index].parameterResultPayloads = payloads
				break
			}
		}
	}
	return nil
}
func inferRuntimeTextParametersFromCalls(program *program, functionName string, parameters []token, textParameters, integerParameters map[string]bool) {
	environment := copyConstantEnvironment(program.constants)
	for _, item := range program.statements {
		if item.kind != "变量" {
			continue
		}
		kind, problem := constantType(item.value, environment)
		if problem == nil {
			environment[item.name.text] = constantValue{kind: kind}
		}
	}
	for index := range program.bindings {
		collectRuntimeTextFunctionCallArguments(program.bindings[index].value, environment, functionName, parameters, textParameters, integerParameters)
	}
	for index := range program.root.fields {
		collectRuntimeTextFunctionCallArguments(program.root.fields[index].value, environment, functionName, parameters, textParameters, integerParameters)
	}
	for index := range program.statements {
		collectRuntimeTextFunctionCallArguments(program.statements[index].value, environment, functionName, parameters, textParameters, integerParameters); collectRuntimeTextFunctionCallArguments(program.statements[index].condition, environment, functionName, parameters, textParameters, integerParameters)
	}
	for index := range program.functions {
		collectRuntimeTextFunctionCallArguments(program.functions[index].body, environment, functionName, parameters, textParameters, integerParameters)
		for statementIndex := range program.functions[index].statements {
			collectRuntimeTextFunctionCallArguments(program.functions[index].statements[statementIndex].value, environment, functionName, parameters, textParameters, integerParameters); collectRuntimeTextFunctionCallArguments(program.functions[index].statements[statementIndex].condition, environment, functionName, parameters, textParameters, integerParameters)
		}
	}
}
func collectRuntimeTextFunctionCallArguments(input value, environment map[string]constantValue, functionName string, parameters []token, textParameters, integerParameters map[string]bool) {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && current.text == functionName {
			for index, argument := range current.args {
				kind, problem := constantType(argument, environment)
				if problem != nil || index >= len(parameters) || kind == "" {
					continue
				}
				parameterName := parameters[index].text
				if kind == "字符串" || kind == "运行时字符串" {
					textParameters[parameterName] = true
				}
				if kind == "整数" || kind == "运行时整数" {
					integerParameters[parameterName] = true
				}
			}
		}
		return true
	})
}
func collectRuntimeIntegerParameterUses(items []statement, parameters, integerParameters map[string]bool) {
	walkStatements(items, func(item statement) bool {
		collectRuntimeIntegerParameterUsesValue(item.value, parameters, integerParameters); collectRuntimeIntegerParameterUsesValue(item.condition, parameters, integerParameters)
		return true
	})
}
func isRuntimeTextValueOperation(name string) bool {
	switch name {
	case "readText", "textLength", "textByteLength", "textByteAt", "textConcat", "substring", "equals", "notEquals", "argumentText", "readFile", "readEnvironment", "environmentNames":
		return true
	}
	return false
}
func collectRuntimeIntegerParameterUsesValue(input value, parameters, integerParameters map[string]bool) {
	walkValue(input, func(current value) bool {
		integerOperation := current.kind == "表达式" && (current.operator == "-" || current.operator == "*" || current.operator == "/" || current.operator == "%" || current.operator == "<<" || current.operator == ">>"); integerAddition := current.kind == "表达式" && current.operator == "+" && current.left != nil && current.right != nil && (current.left.kind == "整数" || current.right.kind == "整数")
		integerComparison := current.kind == "比较" && current.left != nil && current.right != nil && (current.left.kind == "整数" || current.right.kind == "整数"); integerEquality := current.kind == "调用" && (current.text == "equals" || current.text == "notEquals") && len(current.args) == 2 && (current.args[0].kind == "整数" || current.args[1].kind == "整数")
		if integerOperation || integerAddition || integerComparison || integerEquality {
			markRuntimeIntegerParameterReferences(current, parameters, integerParameters)
		}
		if current.kind == "调用" && current.text == "textByteAt" && len(current.args) == 2 && current.args[1].kind == "引用" {
			markRuntimeIntegerParameterReferences(current.args[1], parameters, integerParameters)
		}
		return true
	})
}
func markRuntimeIntegerParameterReferences(input value, parameters, integerParameters map[string]bool) {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && isRuntimeTextValueOperation(current.text) {
			return false
		}
		if current.kind == "引用" && parameters[current.text] {
			integerParameters[current.text] = true
		}
		return true
	})
}
func collectRuntimeTextParameterUses(items []statement, parameters, textParameters map[string]bool) {
	aliases := runtimeTextAliasSources(items); previous := -1
	for previous != len(textParameters) {
		previous = len(textParameters); visitRuntimeTextParameterUses(items, parameters, textParameters, aliases)
	}
}
func collectRuntimeTextReturnParameters(items []statement, parameters, textParameters map[string]bool) {
	sources := map[string][]value{}
	for _, item := range items {
		if item.kind == "返回" {
			if item.value.kind == "引用" {
				markRuntimeTextParameterReferencesResolved(item.value, parameters, textParameters, sources, map[string]bool{})
			}
			return
		}
		if item.kind == "变量" {
			input := item.value
			if input.kind == "引用" && (parameters[input.text] || len(sources[input.text]) != 0) ||
				input.kind == "调用" && (input.text == "textConcat" || input.text == "substring") {
				sources[item.name.text] = []value{input}
			}
		}
	}
}
func visitRuntimeTextParameterUses(items []statement, parameters, textParameters map[string]bool, aliases map[string][]value) {
	walkStatements(items, func(item statement) bool {
		collectRuntimeTextParameterUsesValue(item.value, parameters, textParameters, aliases); collectRuntimeTextParameterUsesValue(item.condition, parameters, textParameters, aliases)
		collectRuntimeTextParameterComparisons(item.value, parameters, textParameters, aliases); collectRuntimeTextParameterComparisons(item.condition, parameters, textParameters, aliases)
		return true
	})
}
func runtimeTextAliasSources(items []statement) map[string][]value {
	aliases := map[string][]value{}
	walkStatements(items, func(item statement) bool {
		if item.kind == "变量" || item.kind == "赋值" {
			aliases[item.name.text] = append(aliases[item.name.text], item.value)
		}
		return true
	})
	return aliases
}
func collectRuntimeTextParameterUsesValue(input value, parameters, textParameters map[string]bool, aliases map[string][]value) {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && current.text == "textLength" && len(current.args) == 1 {
			markRuntimeTextParameterReferencesResolved(current.args[0], parameters, textParameters, aliases, map[string]bool{})
		}
		if current.kind == "调用" && (current.text == "textByteLength" || current.text == "textByteAt") && len(current.args) > 0 && current.args[0].kind == "引用" {
			markRuntimeTextParameterReferencesResolved(current.args[0], parameters, textParameters, aliases, map[string]bool{})
		}
		return true
	})
}
func markRuntimeTextParameterReferencesResolved(input value, parameters, textParameters map[string]bool, aliases map[string][]value, visited map[string]bool) {
	walkValue(input, func(current value) bool {
		if current.kind == "引用" && parameters[current.text] {
			textParameters[current.text] = true
		}
		if current.kind == "引用" && !visited[current.text] {
			if aliasValues, exists := aliases[current.text]; exists {
				visited[current.text] = true
				for _, alias := range aliasValues {
					markRuntimeTextParameterReferencesResolved(alias, parameters, textParameters, aliases, visited)
				}
			}
		}
		return true
	})
}
func collectRuntimeTextParameterComparisons(input value, parameters, textParameters map[string]bool, aliases map[string][]value) {
	walkValue(input, func(current value) bool {
		var left, right *value
		switch current.kind {
		case "调用":
			if (current.text == "equals" || current.text == "notEquals") && len(current.args) == 2 {
				left, right = &current.args[0], &current.args[1]
			}
		case "比较":
			if isRuntimeTextOrderingOperator(current.operator) {
				left, right = current.left, current.right
			}
		}
		if left != nil && right != nil &&
			(runtimeTextComparisonAnchorResolved(*left, parameters, textParameters, aliases, map[string]bool{}) || runtimeTextComparisonAnchorResolved(*right, parameters, textParameters, aliases, map[string]bool{})) {
			markRuntimeTextParameterReferencesResolved(*left, parameters, textParameters, aliases, map[string]bool{}); markRuntimeTextParameterReferencesResolved(*right, parameters, textParameters, aliases, map[string]bool{})
		}
		return true
	})
}
func runtimeTextComparisonAnchorResolved(input value, parameters, textParameters map[string]bool, aliases map[string][]value, visited map[string]bool) bool {
	if input.kind == "引用" && !visited[input.text] {
		if aliasValues, exists := aliases[input.text]; exists {
			visited[input.text] = true
			for _, alias := range aliasValues {
				if runtimeTextComparisonAnchorResolved(alias, parameters, textParameters, aliases, visited) {
					return true
				}
			}
		}
	}
	return runtimeTextComparisonAnchor(input, parameters, textParameters)
}
func runtimeTextComparisonAnchor(input value, parameters, textParameters map[string]bool) bool {
	if input.kind == "字符串" {
		return true
	}
	if input.kind == "引用" {
		return textParameters[input.text] && parameters[input.text]
	}
	if input.kind != "调用" {
		return false
	}
	switch input.text {
	case "readText", "argumentText", "textConcat", "substring":
		return true
	}
	return false
}
func isRuntimeTextOrderingOperator(operator string) bool {
	return operator == "<" || operator == "<=" || operator == ">" || operator == ">="
}
func runtimeFunctionParameterTypes(function functionDecl) []string {
	if len(function.parameterTypes) == len(function.parameters) {
		return function.parameterTypes
	}
	types := make([]string, len(function.parameters))
	for index := range types {
		types[index] = "运行时整数"
	}
	return types
}
func runtimeResultParameterPayload(function functionDecl, name string) string {
	if payload := function.parameterResultPayloads[name]; payload != "" {
		return payload
	}
	return "整数"
}
func runtimeFunctionParameterPayloads(function functionDecl) []string {
	types := runtimeFunctionParameterTypes(function); payloads := make([]string, len(types))
	for index, parameterType := range types {
		if parameterType == "运行时结果" {
			payloads[index] = runtimeResultParameterPayload(function, function.parameters[index].text)
		}
	}
	return payloads
}
func runtimeFunctionRegisterCount(function functionDecl) int {
	payloads := runtimeFunctionParameterPayloads(function); count := 0
	for index, parameterType := range runtimeFunctionParameterTypes(function) {
		if parameterType == "运行时结果" && isTextResultPayload(payloads[index]) {
			count += 4
		} else if parameterType == "运行时字符串" || parameterType == "运行时结果" {
			count += 2
		} else {
			count++
		}
	}
	return count
}
func runtimeFunctionParameterOffsets(types, payloads []string) []uint32 {
	offsets := make([]uint32, len(types)); offset := uint32(0)
	for index := range types {
		offsets[index] = offset; offset += 16
		if types[index] == "运行时结果" && index < len(payloads) && isTextResultPayload(payloads[index]) {
			offset += 16
		}
	}
	return offsets
}
func runtimeFunctionParameterBytes(types, payloads []string) uint32 {
	bytes := uint32(0)
	for index := range types {
		bytes += 16
		if types[index] == "运行时结果" && index < len(payloads) && isTextResultPayload(payloads[index]) {
			bytes += 16
		}
	}
	return bytes
}
func registerIndexForParameter(types, payloads []string, index int) int {
	register := 0
	for i := 0; i < index; i++ {
		if types[i] == "运行时结果" && i < len(payloads) && isTextResultPayload(payloads[i]) {
			register += 4
		} else if types[i] == "运行时字符串" || types[i] == "运行时结果" {
			register += 2
		} else {
			register++
		}
	}
	return register
}
func runtimeStatementLocalType(function functionDecl, name string) string {
	if function.localTypes[name] == "运行时字符串" {
		return "运行时字符串"
	}
	if function.localTypes[name] == "运行时结果" {
		return "运行时结果"
	}
	return "运行时整数"
}
func runtimeStatementLocalSlotCount(function functionDecl) int {
	count := 0
	for _, name := range functionLocalNames(function.statements) {
		if runtimeStatementLocalType(function, name) == "运行时字符串" {
			count += 3
		} else if runtimeStatementLocalType(function, name) == "运行时结果" {
			if isTextResultPayload(function.localResultPayloads[name]) {
				count += 4
			} else {
				count += 2
			}
		} else {
			count++
		}
	}
	return count
}
func isRuntimeTextDurableCopyValue(input value, environment map[string]constantValue) bool {
	kind, problem := constantType(input, environment)
	if problem != nil || kind != "运行时字符串" {
		return false
	}
	if input.kind == "引用" {
		return environment[input.text].kind == "运行时字符串"
	}
	return input.kind == "调用" && (input.text == "textConcat" || input.text == "substring")
}
func classifyRuntimeFunctionLocalTypes(program *program, payloads map[string]constantValue) *issue {
	for _, name := range program.runtimeFunctionNames {
		function := program.runtimeFunctions[name]
		if function.body.kind != "" {
			continue
		}
		environment := copyConstantEnvironment(payloads); parameterPayloads := runtimeFunctionParameterPayloads(function)
		for index, parameter := range function.parameters {
			kind := runtimeFunctionParameterTypes(function)[index]; environment[parameter.text] = constantValue{kind: kind, resultPayload: parameterPayloads[index]}
		}
		localTypes := map[string]string{}; localResultPayloads := map[string]string{}
		resultObservations := map[string]bool{}; var collectResultObservations func([]statement)
		collectResultObservations = func(items []statement) {
			for _, item := range items {
				collectRuntimeResultObservations(item.value, resultObservations); collectRuntimeResultObservations(item.condition, resultObservations)
				collectResultObservations(item.body); collectResultObservations(item.otherwise)
			}
		}
		collectResultObservations(function.statements); var visit func([]statement)
		visit = func(items []statement) {
			for _, item := range items {
				switch item.kind {
				case "变量", "赋值":
					kind, problem := constantType(item.value, environment)
					if problem != nil {
						if item.kind == "变量" && resultObservations[item.name.text] {
							localTypes[item.name.text] = "运行时结果"; localResultPayloads[item.name.text] = "整数"
							environment[item.name.text] = constantValue{kind: "运行时结果", resultPayload: "整数"}
						}
						continue
					}
					if item.kind == "变量" {
						switch kind {
						case "整数", "运行时整数":
							localTypes[item.name.text] = "运行时整数"
						case "运行时字符串":
							localTypes[item.name.text] = "运行时字符串"
						case "运行时结果":
							localTypes[item.name.text] = "运行时结果"
						default:
							localTypes[item.name.text] = kind
						}
						constant := constantValue{kind: localTypes[item.name.text]}
						if kind == "运行时结果" {
							constant.resultPayload = runtimeResultPayload(item.value, environment); localResultPayloads[item.name.text] = constant.resultPayload
						}
						environment[item.name.text] = constant
						continue
					}
					if localTypes[item.name.text] != "" {
						if localTypes[item.name.text] == "运行时字符串" && kind == "运行时字符串" || localTypes[item.name.text] == "运行时整数" && (kind == "整数" || kind == "运行时整数") || localTypes[item.name.text] == "运行时结果" && kind == "运行时结果" {
							environment[item.name.text] = constantValue{kind: localTypes[item.name.text]}
						}
					}
				case "若":
					visit(item.body); visit(item.otherwise)
				}
			}
		}
		visit(function.statements); function.localTypes = localTypes
		function.localResultPayloads = localResultPayloads; program.runtimeFunctions[name] = function
		for index := range program.functions {
			if program.functions[index].name.text == name {
				program.functions[index].localTypes = localTypes; program.functions[index].localResultPayloads = localResultPayloads
				break
			}
		}
	}
	return nil
}
func collectRuntimeResultObservations(input value, observations map[string]bool) {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && (current.text == "是否失败" || current.text == "错误码" || current.text == "结果值") && len(current.args) == 1 && current.args[0].kind == "引用" {
			observations[current.args[0].text] = true
		}
		return true
	})
}
func annotateRuntimeTextReturns(program *program) {
	for _, name := range program.runtimeFunctionNames {
		function := program.runtimeFunctions[name]
		if function.body.kind == "" && function.runtimeReturnsResult && function.runtimeResultPayload == "" {
			function.runtimeResultPayload = "整数"; program.runtimeFunctions[name] = function
			for index := range program.functions {
				if program.functions[index].name.text == name {
					program.functions[index].runtimeResultPayload = "整数"
					break
				}
			}
		}
		if function.body.kind != "" || function.runtimeReturnsResult || len(function.statements) == 0 {
			continue
		}
		last := function.statements[len(function.statements)-1]
		if last.kind != "返回" || last.value.kind != "引用" {
			continue
		}
		if function.localTypes[last.value.text] != "运行时字符串" {
			continue
		}
		if containsStatementKind(function.statements[:len(function.statements)-1], "返回") {
			continue
		}
		function.runtimeReturnsText = true; function.runtimeReturnLocal = last.value.text
		program.runtimeFunctions[name] = function
		for index := range program.functions {
			if program.functions[index].name.text == name {
				program.functions[index].runtimeReturnsText = true; program.functions[index].runtimeReturnLocal = last.value.text
				break
			}
		}
	}
}
func validateRuntimeTextReturnCalls(program *program) *issue {
	for index := range program.functions {
		function := program.functions[index]
		if containsRuntimeTextReturnCall(function.body) {
			return runtimeTextReturnCallIssue(function.name.pos)
		}
		for statementIndex := range function.statements {
			item := &function.statements[statementIndex]
			if statementContainsRuntimeTextReturnCall(*item) {
				return runtimeTextReturnCallIssue(item.name.pos)
			}
		}
	}
	for _, item := range program.bindings {
		if containsRuntimeTextReturnCall(item.value) {
			return runtimeTextReturnCallIssue(item.value.pos)
		}
	}
	for index := range program.statements {
		item := &program.statements[index]
		if !allowsRuntimeTextReturnCall(*item) && statementContainsRuntimeTextReturnCall(*item) {
			return runtimeTextReturnCallIssue(item.name.pos)
		}
	}
	for _, item := range program.statements {
		if item.kind == "变量" || item.kind == "赋值" {
			continue
		}
		if statementContainsRuntimeTextReturnCall(item) {
			return runtimeTextReturnCallIssue(item.name.pos)
		}
	}
	for _, field := range program.root.fields {
		call := field.value
		if field.name.text == "输出" && call.kind == "调用" && call.runtimeFunction && call.runtimeFunctionText && !valuesContainRuntimeTextReturnCall(call.args) {
			continue
		}
		if field.name.text == "输出" && runtimeTextReturnCallAsBorrowedTextArgument(field.value) {
			continue
		}
		if !runtimeTextReturnCallAsBorrowedTextArgument(field.value) && containsRuntimeTextReturnCall(field.value) {
			return runtimeTextReturnCallIssue(field.value.pos)
		}
	}
	return nil
}
func runtimeTextReturnCallIssue(pos sourcePosition) *issue {
	return &issue{code: "S-3023", expected: "运行时文本返回函数仅直接作为程序输出", actual: "文本返回调用出现在变量、条件、函数体或嵌套表达式中", suggestion: "先把文本返回函数调用保留为程序根 输出 的完整表达式", pos: pos}
}
func statementContainsRuntimeTextReturnCall(item statement) bool {
	return containsRuntimeTextReturnCall(item.value) || containsRuntimeTextReturnCall(item.condition) || statementsContainRuntimeTextReturnCall(item.body) || statementsContainRuntimeTextReturnCall(item.otherwise)
}
func isTextReturnCall(input value) bool {
	return input.kind == "调用" && input.runtimeFunction && input.runtimeFunctionText
}
func statementsContainRuntimeTextReturnCall(items []statement) bool {
	for _, item := range items {
		if statementContainsRuntimeTextReturnCall(item) {
			return true
		}
	}
	return false
}
func valuesContainRuntimeTextReturnCall(items []value) bool {
	for _, item := range items {
		if containsRuntimeTextReturnCall(item) {
			return true
		}
	}
	return false
}
func containsRuntimeTextReturnCall(input value) bool {
	found := false
	walkValue(input, func(current value) bool {
		if current.kind == "调用" && current.runtimeFunction && current.runtimeFunctionText {
			found = true
			return false
		}
		return !found
	})
	return found
}
func annotateRuntimeResultPayloads(program *program, constants map[string]constantValue) {
	functions := map[string]functionDecl{}
	for _, function := range program.functions {
		functions[function.name.text] = function
	}
	for {
		changed := false
		for _, name := range program.runtimeFunctionNames {
			function := functions[name]
			if !function.runtimeReturnsResult {
				continue
			}
			environment := copyConstantEnvironment(constants); parameterTypes := runtimeFunctionParameterTypes(function)
			for index, parameter := range function.parameters {
				environment[parameter.text] = constantValue{kind: parameterTypes[index]}
			}
			for local, payload := range function.localResultPayloads {
				environment[local] = constantValue{kind: "运行时结果", resultPayload: payload}
			}
			payload := runtimeResultReturnPayload(function.body, functions, environment, map[string]bool{name: true})
			if len(function.statements) > 0 {
				for _, item := range function.statements {
					if item.kind != "返回" {
						continue
					}
					if fromStatements := runtimeResultReturnPayload(item.value, functions, environment, map[string]bool{name: true}); fromStatements != "" {
						payload = fromStatements
						break
					}
				}
			}
			if payload != "" && payload != function.runtimeResultPayload {
				function.runtimeResultPayload = payload; functions[name] = function
				program.runtimeFunctions[name] = function
				for index := range program.functions {
					if program.functions[index].name.text == name {
						program.functions[index].runtimeResultPayload = payload
					}
				}
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	annotateRuntimeResultCalls(program, program.runtimeFunctions)
}
func runtimeResultReturnPayload(input value, functions map[string]functionDecl, environment map[string]constantValue, visiting map[string]bool) string {
	if input.kind == "调用" {
		if function, exists := functions[input.text]; exists && function.runtimeReturnsResult {
			if function.runtimeResultPayload != "" {
				return function.runtimeResultPayload
			}
			if visiting[input.text] {
				return ""
			}
			visiting[input.text] = true; payload := runtimeResultReturnPayload(function.body, functions, environment, visiting)
			delete(visiting, input.text)
			return payload
		}
		if input.text == "ifElse" && len(input.args) == 3 {
			whenTrue := runtimeResultReturnPayload(input.args[1], functions, environment, visiting); whenFalse := runtimeResultReturnPayload(input.args[2], functions, environment, visiting)
			if whenTrue == "" {
				return whenFalse
			}
			if whenFalse == "" || whenTrue == whenFalse {
				return whenTrue
			}
			if isTextResultPayload(whenTrue) || isTextResultPayload(whenFalse) {
				return "文本"
			}
			return ""
		}
	}
	return runtimeResultPayload(input, environment)
}
func annotateRuntimeResultCalls(program *program, runtimeFunctions map[string]functionDecl) {
	for index := range program.bindings {
		setRuntimeFunctionPayload(&program.bindings[index].value, runtimeFunctions)
	}
	for index := range program.root.fields {
		setRuntimeFunctionPayload(&program.root.fields[index].value, runtimeFunctions)
	}
	for index := range program.statements {
		setStatementRuntimeFunctionPayload(&program.statements[index], runtimeFunctions)
	}
	for index := range program.functions {
		setRuntimeFunctionPayload(&program.functions[index].body, runtimeFunctions)
		for statementIndex := range program.functions[index].statements {
			setStatementRuntimeFunctionPayload(&program.functions[index].statements[statementIndex], runtimeFunctions)
		}
	}
}
func setRuntimeFunctionPayload(input *value, runtimeFunctions map[string]functionDecl) {
	if input == nil {
		return
	}
	if input.kind == "调用" {
		if function, exists := runtimeFunctions[input.text]; exists {
			input.runtimeFunction = true; input.runtimeFunctionResult = function.runtimeReturnsResult
			input.runtimeFunctionPayload = function.runtimeResultPayload; input.runtimeFunctionText = function.runtimeReturnsText
			input.runtimeArgumentTypes = runtimeFunctionParameterTypes(function); input.runtimeArgumentPayloads = runtimeFunctionParameterPayloads(function)
		}
	}
	for index := range input.args {
		setRuntimeFunctionPayload(&input.args[index], runtimeFunctions)
	}
	for index := range input.fields {
		setRuntimeFunctionPayload(&input.fields[index].value, runtimeFunctions)
	}
	if input.left != nil {
		setRuntimeFunctionPayload(input.left, runtimeFunctions)
	}
	if input.right != nil {
		setRuntimeFunctionPayload(input.right, runtimeFunctions)
	}
}
func setStatementRuntimeFunctionPayload(item *statement, runtimeFunctions map[string]functionDecl) {
	setRuntimeFunctionPayload(&item.value, runtimeFunctions); setRuntimeFunctionPayload(&item.condition, runtimeFunctions)
	for index := range item.body {
		setStatementRuntimeFunctionPayload(&item.body[index], runtimeFunctions)
	}
	for index := range item.otherwise {
		setStatementRuntimeFunctionPayload(&item.otherwise[index], runtimeFunctions)
	}
}
func blockFunctionReturnsResult(statements []statement) bool {
	return blockFunctionReturnsResultIn(statements, statements)
}
func blockFunctionReturnsResultValue(statements []statement, value value) bool {
	if runtimeResultReturnShape(value, nil, map[string]bool{}) == "运行时结果" {
		return true
	}
	return value.kind == "引用" && localResultShape(statements, value.text, nil)
}
func blockFunctionReturnsResultIn(statements, allStatements []statement) bool {
	for _, item := range statements {
		if item.kind == "返回" && blockFunctionReturnsResultValue(allStatements, item.value) {
			return true
		}
		if blockFunctionReturnsResultIn(item.body, allStatements) || blockFunctionReturnsResultIn(item.otherwise, allStatements) {
			return true
		}
	}
	return false
}
func functionBodyUsesResultParameter(function functionDecl) bool {
	parameters := map[string]bool{}
	for _, parameter := range function.parameters {
		parameters[parameter.text] = true
	}
	var inspect func(value) bool
	inspect = func(input value) bool {
		if input.kind == "调用" && (input.text == "是否失败" || input.text == "错误码" || input.text == "结果值") && len(input.args) == 1 && input.args[0].kind == "引用" && parameters[input.args[0].text] {
			return true
		}
		for _, argument := range input.args {
			if inspect(argument) {
				return true
			}
		}
		for _, item := range input.fields {
			if inspect(item.value) {
				return true
			}
		}
		if input.left != nil && inspect(*input.left) || input.right != nil && inspect(*input.right) {
			return true
		}
		return false
	}
	if inspect(function.body) {
		return true
	}
	for _, item := range function.statements {
		if inspect(item.value) || inspect(item.condition) {
			return true
		}
	}
	return false
}
func functionReturnsResultParameter(function functionDecl) bool {
	parameters := map[string]bool{}; integerParameters := map[string]bool{}
	for _, parameter := range function.parameters {
		parameters[parameter.text] = true
	}
	collectRuntimeIntegerParameterUses(function.statements, parameters, integerParameters); collectRuntimeIntegerParameterUsesValue(function.body, parameters, integerParameters)
	for name := range parameters {
		if !integerParameters[name] && functionResultBranchUsesParameter(function.body, name) {
			return true
		}
	}
	return false
}
func localResultShape(statements []statement, name string, functions map[string]functionDecl) bool {
	for _, statement := range statements {
		if statement.kind == "变量" && statement.name.text == name && runtimeResultReturnShape(statement.value, functions, map[string]bool{}) == "运行时结果" {
			return true
		}
		if localResultShape(statement.body, name, functions) || localResultShape(statement.otherwise, name, functions) {
			return true
		}
	}
	return false
}
func annotateRuntimeFunctionResults(functions map[string]functionDecl) {
	changed := true
	for changed {
		changed = false
		for name, function := range functions {
			if function.runtimeReturnsResult {
				continue
			}
			if runtimeResultReturnShape(function.body, functions, map[string]bool{name: true}) == "运行时结果" || blockFunctionReturnsResult(function.statements) || functionBodyUsesResultParameter(function) || functionReturnsResultParameter(function) {
				function.runtimeReturnsResult = true; functions[name] = function
				changed = true
			}
		}
	}
}
func runtimeResultReturnShape(input value, functions map[string]functionDecl, visiting map[string]bool) string {
	if input.kind == "引用" {
		if localResultShape(nil, input.text, functions) {
			return "运行时结果"
		}
		for _, function := range functions {
			if localResultShape(function.statements, input.text, functions) {
				return "运行时结果"
			}
		}
	}
	if input.kind == "调用" {
		if function, exists := functions[input.text]; exists {
			if function.runtimeReturnsResult {
				return "运行时结果"
			}
			if visiting[input.text] {
				return ""
			}
			visiting[input.text] = true; shape := runtimeResultReturnShape(function.body, functions, visiting)
			delete(visiting, input.text)
			return shape
		}
		if input.text == "成功" || input.text == "失败" {
			return "运行时结果"
		}
		if input.text == "ifElse" && len(input.args) == 3 {
			whenTrue := runtimeResultReturnShape(input.args[1], functions, visiting); whenFalse := runtimeResultReturnShape(input.args[2], functions, visiting)
			if whenTrue == "运行时结果" && (whenFalse == "运行时结果" || whenFalse == "") || whenFalse == "运行时结果" && whenTrue == "" {
				return "运行时结果"
			}
		}
	}
	return ""
}
func collectFunctionCalls(input value, functions map[string]functionDecl, calls map[string]bool) {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" {
			if _, exists := functions[current.text]; exists {
				calls[current.text] = true
			}
		}
		return true
	})
}
func callGraphReaches(graph map[string]map[string]bool, start, current string, visited map[string]bool) bool {
	for next := range graph[current] {
		if next == start {
			return true
		}
		if visited[next] {
			continue
		}
		visited[next] = true
		if callGraphReaches(graph, start, next, visited) {
			return true
		}
	}
	return false
}
func expandFunctionStatement(item statement, functions map[string]functionDecl, runtimeFunctions map[string]functionDecl, depth int, remaining *int, rootEffects map[string]bool, payloads map[string]constantValue, statements []statement) (statement, *issue) {
	var problem *issue; localPayloads := payloads
	for _, candidate := range statements {
		if candidate.kind != "变量" || candidate.value.kind != "调用" {
			continue
		}
		kind, kindProblem := constantType(candidate.value, localPayloads)
		if kindProblem != nil || kind != "运行时结果" {
			continue
		}
		localPayloads = copyConstantEnvironment(localPayloads); localPayloads[candidate.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(candidate.value, localPayloads)}
	}
	for _, candidate := range statements {
		if candidate.kind != "变量" {
			continue
		}
		kind, kindProblem := constantType(candidate.value, payloads)
		if kindProblem != nil || kind != "运行时结果" {
			continue
		}
		localPayloads = copyConstantEnvironment(localPayloads); localPayloads[candidate.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(candidate.value, localPayloads)}
	}
	item.value, problem = expandFunctionValue(item.value, functions, runtimeFunctions, nil, depth, remaining, rootEffects, localPayloads, statements)
	if problem != nil {
		return statement{}, problem
	}
	item.condition, problem = expandFunctionValue(item.condition, functions, runtimeFunctions, nil, depth, remaining, rootEffects, localPayloads, statements)
	if problem != nil {
		return statement{}, problem
	}
	for index := range item.body {
		item.body[index], problem = expandFunctionStatement(item.body[index], functions, runtimeFunctions, depth+1, remaining, rootEffects, localPayloads, statements)
		if problem != nil {
			return statement{}, problem
		}
	}
	for index := range item.otherwise {
		item.otherwise[index], problem = expandFunctionStatement(item.otherwise[index], functions, runtimeFunctions, depth+1, remaining, rootEffects, localPayloads, statements)
		if problem != nil {
			return statement{}, problem
		}
	}
	return item, nil
}
func expandFunctionValue(input value, functions map[string]functionDecl, runtimeFunctions map[string]functionDecl, stack []string, depth int, remaining *int, rootEffects map[string]bool, payloads map[string]constantValue, statements []statement) (value, *issue) {
	if depth > 256 {
		return value{}, &issue{code: "S-3063", expected: "不超过 256 层的函数表达式", actual: "展开嵌套过深", suggestion: "拆分函数表达式", pos: input.pos}
	}
	callPayloads := payloads
	for _, candidate := range statements {
		if candidate.kind != "变量" || candidate.value.kind != "调用" {
			continue
		}
		kind, kindProblem := constantType(candidate.value, callPayloads)
		if kindProblem != nil || kind != "运行时结果" {
			continue
		}
		callPayloads = copyConstantEnvironment(callPayloads); callPayloads[candidate.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(candidate.value, callPayloads)}
	}
	for _, candidate := range statements {
		if candidate.kind != "变量" {
			continue
		}
		kind, kindProblem := constantType(candidate.value, callPayloads)
		if kindProblem != nil || kind != "运行时结果" {
			continue
		}
		callPayloads = copyConstantEnvironment(callPayloads); callPayloads[candidate.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(candidate.value, callPayloads)}
	}
	if input.kind == "调用" && len(stack) > 0 && isFunctionEffect(input.text) {
		effect := requiredFunctionEffect(input.text)
		for _, active := range stack {
			if !hasFunctionEffect(functions[active], effect) {
				return value{}, functionEffectIssue(input, "为每层调用函数声明 "+effect+" 效果")
			}
		}
		if !rootEffects[effect] {
			return value{}, functionEffectIssue(input, "在程序根能力中声明 "+effect)
		}
	}
	if input.kind == "调用" {
		if function, exists := runtimeFunctions[input.text]; exists {
			if len(input.args) != len(function.parameters) {
				return value{}, &issue{code: "S-3061", expected: strconv.Itoa(len(function.parameters)) + " 个实参", actual: strconv.Itoa(len(input.args)) + " 个实参", suggestion: "按函数声明提供相同数量的参数", pos: input.pos}
			}
			input.runtimeFunction = true; input.runtimeFunctionResult = function.runtimeReturnsResult
			input.runtimeFunctionPayload = function.runtimeResultPayload; var problem *issue
			for index := range input.args {
				input.args[index], problem = expandFunctionValue(input.args[index], functions, runtimeFunctions, stack, depth+1, remaining, rootEffects, callPayloads, statements)
				if problem != nil {
					return value{}, problem
				}
			}
			return input, nil
		}
	}
	if input.kind == "调用" {
		if function, exists := functions[input.text]; exists && !function.runtimeReturnsResult {
			if len(input.args) != len(function.parameters) {
				return value{}, &issue{code: "S-3061", expected: strconv.Itoa(len(function.parameters)) + " 个实参", actual: strconv.Itoa(len(input.args)) + " 个实参", suggestion: "按函数声明提供相同数量的参数", pos: input.pos}
			}
			for _, active := range stack {
				if active == input.text {
					return value{}, &issue{code: "S-3062", expected: "无递归的纯函数调用图", actual: input.text, suggestion: "拆分循环依赖；递归调用约定尚未实现", pos: input.pos}
				}
				for _, effect := range function.effects {
					if !hasFunctionEffect(functions[active], effect) {
						return value{}, &issue{
							code: "S-4106", expected: "显式传递的函数效果",
							actual:     active + " 未声明 " + effect,
							suggestion: "为调用链中的每个函数补入相同的能力声明", pos: input.pos,
						}
					}
					if !rootEffects[effect] {
						return value{}, &issue{
							code: "S-4106", expected: "程序根已声明的调用效果",
							actual:     "程序根未声明 " + effect,
							suggestion: "在程序根能力数组中补入 " + effect, pos: input.pos,
						}
					}
				}
			}
			arguments := make(map[string]value, len(function.parameters))
			for index, parameter := range function.parameters {
				argument := input.args[index]; parameterUses := countFunctionParameter(function.body, parameter.text)
				if parameterUses == 0 {
					return value{}, &issue{code: "S-3065", expected: "函数体使用的参数", actual: parameter.text, suggestion: "删除未使用参数或在返回表达式中使用它", pos: parameter.pos}
				}
				if parameterUses != 1 && containsFunctionEffect(argument, functions) {
					return value{}, &issue{code: "S-4106", expected: "只引用一次的效果函数实参", actual: "效果实参在函数体内被引用 " + strconv.Itoa(parameterUses) + " 次", suggestion: "让纯函数只使用该形参一次，或先在调用方语句中求值并存入变量", pos: argument.pos}
				}
				expandedArgument, problem := expandFunctionValue(argument, functions, runtimeFunctions, stack, depth+1, remaining, rootEffects, callPayloads, statements)
				if problem != nil {
					return value{}, problem
				}
				arguments[parameter.text] = expandedArgument
			}
			if len(stack) >= 64 {
				return value{}, &issue{code: "S-3063", expected: "不超过 64 层的纯函数展开", actual: "函数展开过深", suggestion: "降低纯函数调用深度", pos: input.pos}
			}
			body, problem := cloneFunctionValue(function.body, 0)
			if problem != nil {
				return value{}, problem
			}
			body, problem = substituteFunctionParameters(body, arguments, 0)
			if problem != nil {
				return value{}, problem
			}
			*remaining -= countValueNodes(body)
			if *remaining < 0 {
				return value{}, &issue{code: "S-3063", expected: "不超过 65536 个展开 AST 节点", actual: "纯函数展开超过预算", suggestion: "拆分或减少重复函数展开", pos: input.pos}
			}
			return expandFunctionValue(body, functions, runtimeFunctions, append(stack, input.text), depth+1, remaining, rootEffects, callPayloads, statements)
		}
		if len(stack) > 0 && !isPureFunctionBuiltin(input.text) && !isFunctionEffect(input.text) {
			return value{}, &issue{code: "S-3064", expected: "已声明的纯函数或受支持的纯内建函数", actual: input.text, suggestion: "声明目标函数，或只使用受支持的无副作用内建函数", pos: input.pos}
		}
	}
	if depth > 256 {
		return value{}, &issue{code: "S-3063", expected: "不超过 256 层的函数表达式", actual: "展开嵌套过深", suggestion: "拆分函数表达式", pos: input.pos}
	}
	var problem *issue
	for index := range input.args {
		input.args[index], problem = expandFunctionValue(input.args[index], functions, runtimeFunctions, stack, depth+1, remaining, rootEffects, callPayloads, statements)
		if problem != nil {
			return value{}, problem
		}
	}
	for index := range input.fields {
		input.fields[index].value, problem = expandFunctionValue(input.fields[index].value, functions, runtimeFunctions, stack, depth+1, remaining, rootEffects, callPayloads, statements)
		if problem != nil {
			return value{}, problem
		}
	}
	if input.left != nil {
		left, leftProblem := expandFunctionValue(*input.left, functions, runtimeFunctions, stack, depth+1, remaining, rootEffects, callPayloads, statements)
		if leftProblem != nil {
			return value{}, leftProblem
		}
		input.left = &left
	}
	if input.right != nil {
		right, rightProblem := expandFunctionValue(*input.right, functions, runtimeFunctions, stack, depth+1, remaining, rootEffects, callPayloads, statements)
		if rightProblem != nil {
			return value{}, rightProblem
		}
		input.right = &right
	}
	return input, nil
}
func collectFunctionLocals(statements []statement, locals map[string]bool) {
	for _, item := range statements {
		if item.kind == "变量" {
			locals[item.name.text] = true
		}
		collectFunctionLocals(item.body, locals); collectFunctionLocals(item.otherwise, locals)
	}
}
func lowerFunctionStatements(statements []statement, environment map[string]value, locals map[string]bool, functions map[string]functionDecl, depth int) (value, *issue) {
	if depth > 256 {
		return value{}, &issue{code: "S-3063", expected: "不超过 256 层的函数局部作用域", actual: "局部作用域嵌套过深", suggestion: "拆分函数局部控制语句", pos: sourcePosition{}}
	}
	if len(statements) == 0 {
		return value{}, functionBlockIssue(sourcePosition{}, "每个函数路径都有返回语句", "缺少返回", "在函数块末尾返回一个值")
	}
	item := statements[0]
	switch item.kind {
	case "变量":
		if _, exists := environment[item.name.text]; exists {
			return value{}, functionBlockIssue(item.name.pos, "唯一的函数局部变量名", item.name.text, "使用未被参数或当前作用域占用的局部变量名")
		}
		initial, problem := lowerFunctionValue(item.value, environment, locals, functions, false)
		if problem != nil {
			return value{}, problem
		}
		if isTextReturnCall(initial) {
			return value{}, functionBlockIssue(item.value.pos, "已复制的局部文本", "未复制的文本返回值", "先声明文本局部变量并复制返回值，再在局部作用域使用")
		}
		next := cloneFunctionEnvironment(environment); next[item.name.text] = initial
		return lowerFunctionStatements(statements[1:], next, locals, functions, depth+1)
	case "赋值":
		if _, exists := environment[item.name.text]; !exists {
			return value{}, functionBlockIssue(item.name.pos, "已声明的函数参数或局部变量", item.name.text, "先用变量声明局部值，再进行赋值")
		}
		updated, problem := lowerFunctionValue(item.value, environment, locals, functions, false)
		if problem != nil {
			return value{}, problem
		}
		next := cloneFunctionEnvironment(environment); next[item.name.text] = updated
		return lowerFunctionStatements(statements[1:], next, locals, functions, depth+1)
	case "返回":
		result, problem := lowerFunctionValue(item.value, environment, locals, functions, true)
		if problem != nil {
			return value{}, problem
		}
		if len(statements) != 1 {
			return value{}, functionBlockIssue(statements[1].name.pos, "返回后没有不可达语句", statements[1].kind, "删除返回后的语句或把后续返回移入条件分支")
		}
		return result, nil
	case "若":
		condition, problem := lowerFunctionValue(item.condition, environment, locals, functions, true)
		if problem != nil {
			return value{}, problem
		}
		whenTrue, problem := lowerFunctionStatements(item.body, cloneFunctionEnvironment(environment), locals, functions, depth+1)
		if problem != nil {
			return value{}, problem
		}
		whenFalse := value{}
		if len(item.otherwise) > 0 {
			if len(statements) != 1 {
				return value{}, functionBlockIssue(statements[1].name.pos, "双分支返回后没有不可达语句", statements[1].kind, "删除若否之后的语句")
			}
			whenFalse, problem = lowerFunctionStatements(item.otherwise, cloneFunctionEnvironment(environment), locals, functions, depth+1)
			if problem != nil {
				return value{}, problem
			}
		} else {
			if len(statements) == 1 {
				return value{}, functionBlockIssue(item.name.pos, "若语句两条路径都有返回", "缺少否分支返回", "补入否分支，或在若语句后返回默认值")
			}
			whenFalse, problem = lowerFunctionStatements(statements[1:], environment, locals, functions, depth+1)
			if problem != nil {
				return value{}, problem
			}
		}
		return value{kind: "调用", text: "ifElse", args: []value{condition, whenTrue, whenFalse}, pos: item.name.pos}, nil
	default:
		return value{}, functionBlockIssue(item.name.pos, "函数局部变量、赋值、若或返回语句", item.kind, "使用当前纯函数块支持的语句")
	}
}
func lowerFunctionValue(input value, environment map[string]value, locals map[string]bool, functions map[string]functionDecl, allowEffects bool) (value, *issue) {
	resolved, problem := substituteFunctionParameters(input, environment, 0)
	if problem != nil {
		return value{}, problem
	}
	if problem = validateLocalReferences(resolved, environment, locals); problem != nil {
		return value{}, problem
	}
	if !allowEffects && containsFunctionEffect(resolved, functions) {
		return value{}, functionBlockIssue(input.pos, "可在局部变量初始化中安全复用的纯表达式", "含输入或可变容器效果", "先在返回表达式中调用效果操作，或将效果结果作为函数实参传入")
	}
	return resolved, nil
}
func validateLocalReferences(input value, environment map[string]value, locals map[string]bool) *issue {
	stack := []value{input}
	for len(stack) > 0 {
		last := len(stack) - 1; current := stack[last]
		stack = stack[:last]
		if current.kind == "引用" && locals[current.text] {
			if _, exists := environment[current.text]; !exists {
				return functionBlockIssue(current.pos, "当前作用域已声明的函数局部变量", current.text, "先声明局部变量，或把引用移入其所在分支")
			}
		}
		stack = append(stack, current.args...)
		for _, item := range current.fields {
			stack = append(stack, item.value)
		}
		if current.left != nil {
			stack = append(stack, *current.left)
		}
		if current.right != nil {
			stack = append(stack, *current.right)
		}
	}
	return nil
}
func cloneFunctionEnvironment(input map[string]value) map[string]value {
	output := make(map[string]value, len(input))
	for name, item := range input {
		output[name] = item
	}
	return output
}
func functionBlockIssue(pos sourcePosition, expected, actual, suggestion string) *issue {
	return &issue{code: "S-3068", expected: expected, actual: actual, suggestion: suggestion, pos: pos}
}
func functionEffects(function functionDecl) ([]string, *issue) {
	if !function.hasEffects {
		return nil, nil
	}
	declaration := function.effectDecl
	if declaration.kind != "调用" || declaration.text != "array" {
		return nil, &issue{code: "S-4105", expected: "静态文本数组效果声明", actual: declaration.kind, suggestion: "使用能力=array（『stdin』）声明函数效果", pos: declaration.pos}
	}
	effects := make([]string, 0, len(declaration.args)); seen := map[string]bool{}
	for _, item := range declaration.args {
		if item.kind != "字符串" {
			return nil, &issue{code: "S-4105", expected: "静态文本效果名", actual: item.kind, suggestion: "仅使用文本字面量声明函数效果", pos: item.pos}
		}
		if item.text != "stdin" && item.text != "filesystem.read" && item.text != "filesystem.write" && item.text != "array.write" && item.text != "map.write" && item.text != "args" {
			return nil, &issue{code: "S-4103", expected: "函数支持的 stdin、filesystem.read、filesystem.write、array.write、map.write 或 args 效果", actual: item.text, suggestion: "当前函数效果子集支持 stdin、filesystem.read、filesystem.write、array.write、map.write 和 args", pos: item.pos}
		}
		if seen[item.text] {
			return nil, &issue{code: "S-4104", expected: "唯一函数效果名", actual: item.text, suggestion: "每项效果只声明一次", pos: item.pos}
		}
		seen[item.text] = true; effects = append(effects, item.text)
	}
	return effects, nil
}
func hasFunctionEffect(function functionDecl, effect string) bool {
	for _, declared := range function.effects {
		if declared == effect {
			return true
		}
	}
	return false
}
func functionEffectIssue(input value, suggestion string) *issue {
	return &issue{code: "S-4106", expected: "显式声明且逐层传递的函数效果", actual: input.text, suggestion: suggestion, pos: input.pos}
}
func substituteFunctionParameters(input value, parameters map[string]value, depth int) (value, *issue) {
	if depth > 256 {
		return value{}, &issue{code: "S-3063", expected: "不超过 256 层的函数表达式", actual: "参数替换嵌套过深", suggestion: "拆分函数表达式", pos: input.pos}
	}
	if input.kind == "引用" {
		if argument, exists := parameters[input.text]; exists {
			return cloneFunctionValue(argument, depth+1)
		}
	}
	var problem *issue
	for index := range input.args {
		input.args[index], problem = substituteFunctionParameters(input.args[index], parameters, depth+1)
		if problem != nil {
			return value{}, problem
		}
	}
	for index := range input.fields {
		input.fields[index].value, problem = substituteFunctionParameters(input.fields[index].value, parameters, depth+1)
		if problem != nil {
			return value{}, problem
		}
	}
	if input.left != nil {
		left, leftProblem := substituteFunctionParameters(*input.left, parameters, depth+1)
		if leftProblem != nil {
			return value{}, leftProblem
		}
		input.left = &left
	}
	if input.right != nil {
		right, rightProblem := substituteFunctionParameters(*input.right, parameters, depth+1)
		if rightProblem != nil {
			return value{}, rightProblem
		}
		input.right = &right
	}
	return input, nil
}
func cloneFunctionValue(input value, depth int) (value, *issue) {
	if depth > 256 {
		return value{}, &issue{code: "S-3063", expected: "不超过 256 层的函数表达式", actual: "克隆函数实参时嵌套过深", suggestion: "拆分函数表达式", pos: input.pos}
	}
	input.args = append([]value(nil), input.args...)
	for index := range input.args {
		cloned, problem := cloneFunctionValue(input.args[index], depth+1)
		if problem != nil {
			return value{}, problem
		}
		input.args[index] = cloned
	}
	input.fields = append([]field(nil), input.fields...)
	for index := range input.fields {
		cloned, problem := cloneFunctionValue(input.fields[index].value, depth+1)
		if problem != nil {
			return value{}, problem
		}
		input.fields[index].value = cloned
	}
	if input.left != nil {
		left, problem := cloneFunctionValue(*input.left, depth+1)
		if problem != nil {
			return value{}, problem
		}
		input.left = &left
	}
	if input.right != nil {
		right, problem := cloneFunctionValue(*input.right, depth+1)
		if problem != nil {
			return value{}, problem
		}
		input.right = &right
	}
	return input, nil
}
func countFunctionParameter(input value, name string) int {
	count := 0
	walkValue(input, func(current value) bool {
		if current.kind == "引用" && current.text == name {
			count++
		}
		return true
	})
	return count
}
func countValueNodes(input value) int {
	count := 0
	walkValue(input, func(value) bool {
		count++
		return true
	})
	return count
}
func isFunctionEffect(name string) bool {
	switch name {
	case "readInteger", "readText", "readFile", "readModule", "readModuleResult", "fileSize", "exists", "isDirectory", "writeFile", "appendFile", "writeByteArray", "removeFile", "renameFile", "createDirectory", "removeDirectory", "setFileMode", "changeFileMode", "replaceFile", "environmentNames", "arrayPush", "arraySet", "映射置", "映射删", "argumentCount", "argumentText", "readEnvironment":
		return true
	}
	return false
}
func requiredFunctionEffect(name string) string {
	if name == "arrayPush" || name == "arraySet" {
		return "array.write"
	}
	if name == "映射置" || name == "映射删" {
		return "map.write"
	}
	if name == "argumentCount" || name == "argumentText" || name == "readEnvironment" || name == "environmentNames" {
		return "args"
	}
	if name == "readFile" || name == "readModule" || name == "readModuleResult" || name == "fileSize" || name == "exists" || name == "isDirectory" {
		return "filesystem.read"
	}
	if name == "writeFile" || name == "appendFile" || name == "writeByteArray" || name == "removeFile" || name == "renameFile" || name == "replaceFile" || name == "createDirectory" || name == "removeDirectory" || name == "setFileMode" || name == "changeFileMode" {
		return "filesystem.write"
	}
	return "stdin"
}
func containsFunctionEffect(input value, functions map[string]functionDecl) bool {
	found := false
	walkValue(input, func(current value) bool {
		if current.kind == "调用" {
			if isFunctionEffect(current.text) {
				found = true
				return false
			}
			if function, exists := functions[current.text]; exists && len(function.effects) > 0 {
				found = true
				return false
			}
		}
		return !found
	})
	return found
}
func isPureFunctionBuiltin(name string) bool {
	switch name {
	case "textConcat", "substring", "textLength", "textByteLength", "textByteAt", "equals", "notEquals", "ifElse", "array", "arrayGet", "arrayLength", "bitAnd", "bitOr", "bitXor", "成功", "失败", "是否失败", "结果值", "错误码", "映射", "映射取", "映射有", "映射长", "映射删":
		return true
	}
	return false
}
func validateRuntimeTextParameterReturns(program *program) *issue {
	for _, name := range program.runtimeFunctionNames {
		function := program.runtimeFunctions[name]; parameterTypes := runtimeFunctionParameterTypes(function)
		parameters := map[string]string{}
		for index, parameter := range function.parameters {
			if index < len(parameterTypes) {
				parameters[parameter.text] = parameterTypes[index]
			}
		}
		var visit func([]statement) *issue
		visit = func(items []statement) *issue {
			for _, item := range items {
				if item.kind == "返回" && item.value.kind == "引用" && parameters[item.value.text] == "运行时字符串" {
					return &issue{code: "S-3068", expected: "文本局部的独立副本", actual: "直接返回文本参数 " + item.value.text, suggestion: "先把文本参数复制到顶层文本局部，再返回该局部", pos: item.value.pos}
				}
				if problem := visit(item.body); problem != nil {
					return problem
				}
				if problem := visit(item.otherwise); problem != nil {
					return problem
				}
			}
			return nil
		}
		if problem := visit(function.statements); problem != nil {
			return problem
		}
	}
	return nil
}
func validateProgram(program *program) *issue {
	payloads := seedRuntimeResultPayloads(program)
	if problem := expandProgramFunctions(program, payloads); problem != nil {
		return problem
	}
	if program.root.kind != "记录" {
		return &issue{code: "S-3001", expected: "程序记录值", actual: program.root.kind, suggestion: "用窗口=【标题=...,文本=...】或程序=【输出=...】描述程序", pos: program.root.pos}
	}
	constants := map[string]constantValue{}
	for _, item := range program.bindings {
		if _, exists := constants[item.name.text]; exists {
			return &issue{code: "S-3007", expected: "唯一常量名", actual: item.name.text, suggestion: "为每个常量使用不同名称", pos: item.name.pos}
		}
		kind, problem := constantType(item.value, constants)
		if problem != nil {
			return problem
		}
		if kind == "运行时整数" {
			return &issue{code: "S-3042", expected: "编译期常量表达式", actual: "readInteger（）运行时值", suggestion: "只在程序根绑定的输出表达式中使用 readInteger", pos: item.value.pos}
		}
		if kind == "运行时整数数组" {
			return &issue{code: "S-3031", expected: "编译期整数数组", actual: "含运行时值的整数数组", suggestion: "将 readInteger（）移到程序中的可变数组初始化", pos: item.value.pos}
		}
		if kind == "运行时字符串数组" {
			return &issue{code: "S-3031", expected: "编译期字符串数组", actual: "含运行时值的字符串数组", suggestion: "将运行时文本数组放入程序语句", pos: item.value.pos}
		}
		if kind == "运行时字符串" {
			return &issue{code: "S-3042", expected: "编译期常量表达式", actual: "readText（）运行时文本", suggestion: "只在程序根绑定的输出表达式中使用 readText", pos: item.value.pos}
		}
		resolved := constantValue{kind: kind}
		if kind == "字符串" {
			resolved.text, problem = evaluateString(item.value, 0, constants)
		} else if kind == "整数" {
			resolved.integer, problem = integerValue(item.value, constants)
		} else {
			resolved.array, problem = evaluateArray(item.value, constants)
		}
		if problem != nil {
			return problem
		}
		constants[item.name.text] = resolved
	}
	program.constants = constants
	if problem := classifyRuntimeFunctionParameterTypes(program); problem != nil {
		return problem
	}
	if problem := classifyRuntimeFunctionLocalTypes(program, constants); problem != nil {
		return problem
	}
	annotateRuntimeTextReturns(program); annotateRuntimeResultPayloads(program, constants)
	if problem := validateRuntimeTextReturnCalls(program); problem != nil {
		return problem
	}
	if problem := validateRuntimeFunctions(program, constants, programRootEffects(program.root)); problem != nil {
		return problem
	}
	if problem := validateRuntimeTextParameterReturns(program); problem != nil {
		return problem
	}
	if program.rootName.text == "程序" {
		if problem := validateConsoleProgram(program.root, constants, program.statements); problem != nil {
			return problem
		}
		return validateConsoleCapabilities(program.root, program.statements, constants)
	}
	if program.rootName.text != "窗口" {
		return &issue{code: "S-2002", expected: "窗口 或 程序", actual: program.rootName.text, suggestion: "使用受支持的程序根绑定", pos: program.rootName.pos}
	}
	if len(program.statements) != 0 {
		return &issue{code: "S-3003", expected: "程序 根绑定中的运行时变量语句", actual: "窗口 根绑定", suggestion: "将变量语句移到程序根绑定中", pos: program.statements[0].name.pos}
	}
	seen := map[string]bool{}
	for _, item := range program.root.fields {
		name := item.name.text
		if seen[name] {
			return &issue{code: "S-3002", expected: "唯一字段", actual: name, suggestion: "删除重复字段", pos: item.name.pos}
		}
		seen[name] = true
		if name != "标题" && name != "文本" {
			return &issue{code: "S-3003", expected: "标题 或 文本", actual: name, suggestion: "删除 P0b 窗口子集不支持的字段", pos: item.name.pos}
		}
		if _, problem := evaluateString(item.value, 0, constants); problem != nil {
			return problem
		}
	}
	for _, name := range []string{"标题", "文本"} {
		if !seen[name] {
			return &issue{code: "S-3006", expected: name + " 字段", actual: "字段缺失", suggestion: "在窗口记录中补入 " + name + " 字符串字段", pos: program.root.pos}
		}
	}
	return nil
}
func annotateResultStateCallsInValue(input *value, constants map[string]constantValue) {
	if input == nil {
		return
	}
	if input.kind == "调用" && (input.text == "是否失败" || input.text == "错误码" || input.text == "结果值") && len(input.args) == 1 {
		argument := input.args[0]
		if argument.kind == "调用" && argument.text == "ifElse" && len(argument.args) == 3 {
			argument.runtimeFunction = true; argument.runtimeFunctionResult = true
			argument.runtimeFunctionPayload = "整数"; argument.runtimeFunctionText = false
			argument.runtimeArgumentTypes = []string{"运行时布尔值", "运行时结果", "运行时结果"}; argument.runtimeArgumentPayloads = []string{"", "整数", "整数"}
			for index := range argument.args {
				annotateResultStateCallsInValue(&argument.args[index], constants)
			}
			return
		}
	}
	for index := range input.args {
		annotateResultStateCallsInValue(&input.args[index], constants)
	}
	for index := range input.fields {
		annotateResultStateCallsInValue(&input.fields[index].value, constants)
	}
	annotateResultStateCallsInValue(input.left, constants); annotateResultStateCallsInValue(input.right, constants)
}
func annotateResultStateCallsInStatements(items []statement, constants map[string]constantValue) {
	for index := range items {
		annotateResultStateCallsInValue(&items[index].value, constants); annotateResultStateCallsInValue(&items[index].condition, constants)
		annotateResultStateCallsInStatements(items[index].body, constants); annotateResultStateCallsInStatements(items[index].otherwise, constants)
	}
}
func statementRuntimeEffects(input value, functions map[string]functionDecl) []string {
	effects := []string{}; seen := map[string]bool{}
	var walk func(value, map[string]bool)
	walk = func(current value, active map[string]bool) {
		if current.kind == "调用" {
			if isFunctionEffect(current.text) {
				effect := requiredFunctionEffect(current.text)
				if !seen[effect] {
					seen[effect] = true; effects = append(effects, effect)
				}
			}
			if function, exists := functions[current.text]; exists && !active[current.text] {
				next := map[string]bool{current.text: true}
				for name := range active {
					next[name] = true
				}
				for _, declared := range function.effects {
					if !seen[declared] {
						seen[declared] = true; effects = append(effects, declared)
					}
				}
				walk(function.body, next)
				for _, item := range function.statements {
					walk(item.value, next); walk(item.condition, next)
				}
			}
		}
		for _, argument := range current.args {
			walk(argument, active)
		}
		for _, field := range current.fields {
			walk(field.value, active)
		}
		if current.left != nil {
			walk(*current.left, active)
		}
		if current.right != nil {
			walk(*current.right, active)
		}
	}
	walk(input, map[string]bool{})
	return effects
}
func hasStatementEffectSupport(input value, functions map[string]functionDecl, allowReadText ...bool) bool {
	if input.kind == "调用" {
		if input.text == "readInteger" || len(allowReadText) > 0 && input.text == "readText" {
			return true
		}
		if isFunctionEffect(input.text) && input.text != "readInteger" {
			return false
		}
		if function, exists := functions[input.text]; exists && len(function.effects) > 0 {
			return false
		}
	}
	for _, argument := range input.args {
		if !hasStatementEffectSupport(argument, functions, allowReadText...) {
			return false
		}
	}
	for _, field := range input.fields {
		if !hasStatementEffectSupport(field.value, functions, allowReadText...) {
			return false
		}
	}
	if input.left != nil && !hasStatementEffectSupport(*input.left, functions, allowReadText...) {
		return false
	}
	if input.right != nil && !hasStatementEffectSupport(*input.right, functions, allowReadText...) {
		return false
	}
	return true
}
func hasRuntimeResultEffectSupport(input value, functions map[string]functionDecl) bool {
	if input.kind == "调用" && input.text == "成功" && len(input.args) == 1 {
		return hasStatementEffectSupport(input.args[0], functions, true)
	}
	if input.kind == "调用" && input.text == "ifElse" && len(input.args) == 3 {
		return hasStatementEffectSupport(input.args[0], functions) &&
			hasRuntimeResultEffectSupport(input.args[1], functions) &&
			hasRuntimeResultEffectSupport(input.args[2], functions)
	}
	return hasStatementEffectSupport(input, functions)
}
func statementEffectSupport(function functionDecl, item statement, input value, functions map[string]functionDecl) bool {
	if (item.kind == "变量" || item.kind == "赋值") && runtimeStatementLocalType(function, item.name.text) == "运行时结果" {
		return hasRuntimeResultEffectSupport(input, functions)
	}
	return hasStatementEffectSupport(input, functions)
}
func validateRuntimeFunctions(program *program, constants map[string]constantValue, rootEffects map[string]bool) *issue {
	functions := make(map[string]functionDecl, len(program.functions))
	for _, function := range program.functions {
		functions[function.name.text] = function
	}
	for _, name := range program.runtimeFunctionNames {
		function := program.runtimeFunctions[name]; environment := copyConstantEnvironment(constants)
		registerCount := runtimeFunctionRegisterCount(function)
		if registerCount > 8 {
			return &issue{code: "S-3062", expected: "最多 8 个展平运行时参数寄存器", actual: strconv.Itoa(registerCount), suggestion: "减少参数；整数参数占一个寄存器，文本参数占两个，文本结果参数占四个", pos: function.name.pos}
		}
		parameterTypes := runtimeFunctionParameterTypes(function); parameterPayloads := runtimeFunctionParameterPayloads(function)
		for index, parameter := range function.parameters {
			environment[parameter.text] = constantValue{kind: parameterTypes[index]}
			if parameterTypes[index] == "运行时结果" {
				environment[parameter.text] = constantValue{kind: parameterTypes[index], resultPayload: parameterPayloads[index]}
			}
		}
		if function.body.kind == "" {
			var registerResultLocals func([]statement)
			registerResultLocals = func(items []statement) {
				for _, item := range items {
					if item.kind == "变量" && runtimeStatementLocalType(function, item.name.text) == "运行时结果" {
						kind, problem := constantType(item.value, environment)
						if problem == nil && kind == "运行时结果" {
							environment[item.name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(item.value, environment)}
						}
					}
					registerResultLocals(item.body); registerResultLocals(item.otherwise)
				}
			}
			registerResultLocals(function.statements); annotateResultStateCallsInStatements(function.statements, environment)
			if problem := validateRuntimeStatementFunction(function, constants, functions, rootEffects); problem != nil {
				return problem
			}
			continue
		}
		kind, problem := constantType(function.body, environment)
		if problem != nil {
			return problem
		}
		if kind == "运行时结果" || isRuntimeResultValue(function.body) {
			payload := runtimeResultPayload(function.body, environment)
			if payload == "" {
				payload = "整数"
			}
			if payload == "整数" || payload == "文本" {
				continue
			}
			if payload == "运行时文本" {
				return &issue{code: "S-3023", expected: "递归运行时函数的整数或编译期文本结果载荷", actual: payload, suggestion: "先把运行时文本结果载荷限制到非递归函数，或改为编译期文本", pos: function.name.pos}
			}
		}
		if kind != "整数" && kind != "运行时整数" {
			return &issue{code: "S-3062", expected: "递归运行时函数的整数返回值", actual: kind, suggestion: "递归运行时函数暂只支持纯整数表达式和整数参数", pos: function.name.pos}
		}
	}
	return nil
}
func validateRuntimeStatementFunction(function functionDecl, constants map[string]constantValue, functions map[string]functionDecl, rootEffects map[string]bool) *issue {
	environment := copyConstantEnvironment(constants); writable := map[string]bool{}
	parameterTypes := runtimeFunctionParameterTypes(function); parameterPayloads := runtimeFunctionParameterPayloads(function)
	parameterSlotCount := 0
	for index, parameter := range function.parameters {
		if parameterTypes[index] == "运行时结果" {
			environment[parameter.text] = constantValue{kind: "运行时结果", resultPayload: parameterPayloads[index]}
			if isTextResultPayload(parameterPayloads[index]) {
				parameterSlotCount += 4
			} else {
				parameterSlotCount += 2
			}
			continue
		}
		environment[parameter.text] = constantValue{kind: parameterTypes[index]}
		if parameterTypes[index] == "运行时字符串" {
			parameterSlotCount += 2
		} else {
			parameterSlotCount++
			writable[parameter.text] = true
		}
	}
	localNames := functionLocalNames(function.statements); seenLocals := map[string]bool{}
	for _, name := range localNames {
		if _, exists := environment[name]; exists || seenLocals[name] {
			return functionBlockIssue(function.name.pos, "函数内唯一的局部变量名", name, "避免局部变量重名，也不得与参数或常量重名")
		}
		seenLocals[name] = true
	}
	localSlotCount := runtimeStatementLocalSlotCount(function)
	for index, item := range function.statements {
		values := []value{item.value, item.condition}
		for _, input := range values {
			for _, required := range statementRuntimeEffects(input, functions) {
				if !hasFunctionEffect(function, required) {
					return &issue{code: "S-4106", expected: "为当前函数声明 " + required + " 效果", actual: required, suggestion: "为当前函数和调用链声明 " + required, pos: item.name.pos}
				}
				if !rootEffects[required] {
					return &issue{code: "S-4106", expected: "程序根能力声明 " + required, actual: required, suggestion: "在程序根能力数组中声明 " + required, pos: item.name.pos}
				}
			}
			if input.kind != "" && !statementEffectSupport(function, item, input, functions) {
				return &issue{code: "S-4106", expected: "受支持的运行时函数效果", actual: "函数体表达式含未支持的效果", suggestion: "把效果求值移到受支持的运行时语句或由调用方传入", pos: input.pos}
			}
		}
		switch item.kind {
		case "变量":
			if _, exists := environment[item.name.text]; exists {
				return functionBlockIssue(item.name.pos, "唯一的函数局部变量名", item.name.text, "使用未被参数或常量占用的局部变量名")
			}
			if !statementEffectSupport(function, item, item.value, functions) {
				return &issue{code: "S-4106", expected: "受支持的局部变量初值效果", actual: "含未支持的效果", suggestion: "通过函数返回表达式或受支持的运行时语句求值", pos: item.value.pos}
			}
			if isTextReturnCall(item.value) {
				environment[item.name.text] = constantValue{kind: "运行时字符串"}; writable[item.name.text] = true
				break
			}
			if runtimeStatementLocalType(function, item.name.text) == "运行时结果" {
				kind, problem := constantType(item.value, environment)
				if problem != nil {
					return problem
				}
				if kind != "运行时结果" {
					return functionBlockIssue(item.value.pos, "函数结果局部初值", kind, "使用返回结果的调用初始化结果局部")
				}
				environment[item.name.text] = constantValue{kind: "运行时结果", resultPayload: runtimeResultPayload(item.value, environment)}; writable[item.name.text] = true
				break
			}
			kind, problem := constantType(item.value, environment)
			if problem != nil {
				return problem
			}
			if runtimeStatementLocalType(function, item.name.text) == "运行时字符串" {
				if kind != "运行时字符串" || !isRuntimeTextDurableCopyValue(item.value, environment) {
					return functionBlockIssue(item.value.pos, "函数文本局部的独立副本", kind, "使用文本参数、文本局部、textConcat 或 substring 结果初始化文本局部")
				}
				environment[item.name.text] = constantValue{kind: "运行时字符串"}; writable[item.name.text] = true
				break
			}
			if kind != "整数" && kind != "运行时整数" {
				return functionBlockIssue(item.value.pos, "整数函数局部初值", kind, "运行时函数顶层局部可保存整数或文本参数的独立副本")
			}
			environment[item.name.text] = constantValue{kind: "运行时整数"}; writable[item.name.text] = true
		case "赋值":
			if !writable[item.name.text] {
				return functionBlockIssue(item.name.pos, "已声明的函数参数或局部变量", item.name.text, "先声明整数局部变量，再进行赋值")
			}
			kind, problem := constantType(item.value, environment)
			if problem != nil {
				return problem
			}
			if environment[item.name.text].kind == "运行时结果" {
				if kind != "运行时结果" {
					return functionBlockIssue(item.value.pos, "函数结果局部赋值", kind, "使用成功、失败或返回结果的调用更新结果局部")
				}
				break
			}
			if environment[item.name.text].kind == "运行时字符串" {
				if kind != "运行时字符串" || !isRuntimeTextDurableCopyValue(item.value, environment) {
					return functionBlockIssue(item.value.pos, "文本局部的独立副本赋值", kind, "从文本参数、文本局部、textConcat 或 substring 结果更新文本局部")
				}
				break
			}
			if kind != "整数" && kind != "运行时整数" {
				return functionBlockIssue(item.value.pos, "整数函数赋值", kind, "运行时循环函数当前只更新整数状态")
			}
		case "若":
			kind, problem := constantType(item.condition, environment)
			if problem != nil {
				return problem
			}
			if kind != "布尔值" && kind != "运行时布尔值" {
				return functionBlockIssue(item.condition.pos, "若语句的布尔条件", kind, "若条件使用整数比较、文本比较或逻辑组合")
			}
			if problem := validateRuntimeLoopBody(item.body, copyConstantEnvironment(environment), copyWritableVariables(writable), functions, rootEffects, function, 0, true); problem != nil {
				return problem
			}
			if problem := validateRuntimeLoopBody(item.otherwise, copyConstantEnvironment(environment), copyWritableVariables(writable), functions, rootEffects, function, 0, true); problem != nil {
				return problem
			}
		case "凡":
			kind, problem := constantType(item.condition, environment)
			if problem != nil {
				return problem
			}
			if kind != "布尔值" && kind != "运行时布尔值" {
				return functionBlockIssue(item.condition.pos, "凡循环的布尔条件", kind, "在凡条件中使用整数比较、文本比较或逻辑组合")
			}
			if kind == "布尔值" {
				selected, problem := evaluateBoolean(item.condition, 0, environment)
				if problem != nil {
					return problem
				}
				if selected {
					return functionBlockIssue(item.name.pos, "可终止的凡循环条件", "编译期为真", "使用依赖整数状态的运行时终止条件")
				}
			}
			if len(item.body) == 0 {
				return functionBlockIssue(item.name.pos, "非空的凡循环体", "空循环体", "在循环体内更新至少一个整数状态")
			}
			if problem := validateRuntimeLoopBody(item.body, copyConstantEnvironment(environment), copyWritableVariables(writable), functions, rootEffects, function, 1, false); problem != nil {
				return problem
			}
		case "返回":
			if index != len(function.statements)-1 {
				return functionBlockIssue(function.statements[index+1].name.pos, "返回后没有不可达语句", function.statements[index+1].kind, "把返回保留为函数块最后一条语句")
			}
			kind, problem := constantType(item.value, environment)
			if problem != nil {
				return problem
			}
			if function.runtimeReturnsText {
				if item.value.kind != "引用" || item.value.text != function.runtimeReturnLocal || kind != "运行时字符串" {
					return functionBlockIssue(item.value.pos, "已初始化函数文本局部的所有权返回", kind, "最后直接返回一个已初始化的顶层文本局部")
				}
			} else if kind != "整数" && kind != "运行时整数" && !(kind == "运行时结果" && function.runtimeReturnsResult) && false {
				return functionBlockIssue(item.value.pos, "整数运行时循环函数返回值", kind, "当前只开放无效果整数函数块内的凡循环、结果局部或顶层文本局部返回")
			}
			stateSlots := parameterSlotCount + localSlotCount
			if stateSlots > 502 {
				return &issue{code: "S-3040", expected: "不超过 502 个运行时函数状态槽", actual: strconv.Itoa(stateSlots), suggestion: "减少函数参数和局部状态；文本参数占指针与长度两个槽，文本局部另占容量槽", pos: function.name.pos}
			}
			return nil
		default:
			return functionBlockIssue(item.name.pos, "变量、赋值、若、凡或返回语句", item.kind, "顶层使用变量、赋值、若、凡或返回；循环控制放在循环体内")
		}
	}
	if runtimeStatementsAlwaysReturn(function.statements) {
		return nil
	}
	return functionBlockIssue(function.name.pos, "函数块末尾返回整数状态", "缺少返回", "在凡循环之后返回结果")
}
func runtimeStatementsAlwaysReturn(items []statement) bool {
	if len(items) == 0 {
		return false
	}
	last := items[len(items)-1]
	switch last.kind {
	case "返回":
		return true
	case "若":
		if len(last.otherwise) == 0 {
			return false
		}
		return runtimeStatementsAlwaysReturn(last.body) && runtimeStatementsAlwaysReturn(last.otherwise)
	}
	return false
}
func validateRuntimeLoopBody(items []statement, environment map[string]constantValue, writable map[string]bool, functions map[string]functionDecl, rootEffects map[string]bool, function functionDecl, loopDepth int, allowResultReturn bool) *issue {
	for index, item := range items {
		for _, input := range []value{item.value, item.condition} {
			if input.kind == "" {
				continue
			}
			for _, required := range statementRuntimeEffects(input, functions) {
				if !hasFunctionEffect(function, required) {
					return &issue{code: "S-4106", expected: "为循环所在函数声明 " + required + " 效果", actual: required, suggestion: "在函数能力数组中声明该效果", pos: item.name.pos}
				}
				if !rootEffects[required] {
					return &issue{code: "S-4106", expected: "程序根能力声明 " + required, actual: required, suggestion: "在程序根能力数组中声明该效果", pos: item.name.pos}
				}
			}
			if !hasStatementEffectSupport(input, functions) {
				return &issue{code: "S-4106", expected: "受支持的运行时循环体效果", actual: "循环体表达式含未支持的效果", suggestion: "把效果求值移到受支持的运行时语句或由调用方传入", pos: input.pos}
			}
		}
		switch item.kind {
		case "变量":
			if _, exists := environment[item.name.text]; exists {
				return functionBlockIssue(item.name.pos, "块内唯一的整数局部变量名", item.name.text, "避免局部变量与可见名称重名")
			}
			kind, problem := constantType(item.value, environment)
			if problem != nil {
				return problem
			}
			if kind != "整数" && kind != "运行时整数" {
				return functionBlockIssue(item.value.pos, "整数循环体局部初值", kind, "循环体局部变量当前只保存整数状态")
			}
			environment[item.name.text] = constantValue{kind: "运行时整数"}; writable[item.name.text] = true
		case "赋值":
			if !writable[item.name.text] {
				return functionBlockIssue(item.name.pos, "已声明的函数参数或局部变量", item.name.text, "循环体只能更新已声明整数状态")
			}
			kind, problem := constantType(item.value, environment)
			if problem != nil {
				return problem
			}
			if kind != "整数" && kind != "运行时整数" {
				return functionBlockIssue(item.value.pos, "整数循环体赋值", kind, "循环体当前只更新整数状态")
			}
		case "若":
			kind, problem := constantType(item.condition, environment)
			if problem != nil {
				return problem
			}
			if kind != "布尔值" && kind != "运行时布尔值" {
				return functionBlockIssue(item.condition.pos, "若语句的布尔条件", kind, "若条件使用整数比较、文本比较或逻辑组合")
			}
			if problem := validateRuntimeLoopBody(item.body, copyConstantEnvironment(environment), copyWritableVariables(writable), functions, rootEffects, function, loopDepth, allowResultReturn); problem != nil {
				return problem
			}
			if problem := validateRuntimeLoopBody(item.otherwise, copyConstantEnvironment(environment), copyWritableVariables(writable), functions, rootEffects, function, loopDepth, allowResultReturn); problem != nil {
				return problem
			}
		case "凡":
			kind, problem := constantType(item.condition, environment)
			if problem != nil {
				return problem
			}
			if kind != "布尔值" && kind != "运行时布尔值" {
				return functionBlockIssue(item.condition.pos, "凡循环的布尔条件", kind, "凡条件使用整数比较、文本比较或逻辑组合")
			}
			if kind == "布尔值" {
				selected, problem := evaluateBoolean(item.condition, 0, environment)
				if problem != nil {
					return problem
				}
				if selected {
					return functionBlockIssue(item.name.pos, "可终止的嵌套凡循环条件", "编译期为真", "使用依赖整数状态的运行时终止条件")
				}
			}
			if len(item.body) == 0 {
				return functionBlockIssue(item.name.pos, "非空的嵌套凡循环体", "空循环体", "在嵌套循环体内更新整数状态或跳出")
			}
			if problem := validateRuntimeLoopBody(item.body, copyConstantEnvironment(environment), copyWritableVariables(writable), functions, rootEffects, function, loopDepth+1, false); problem != nil {
				return problem
			}
		case "跳出", "继续":
			if loopDepth == 0 {
				return functionBlockIssue(item.name.pos, "凡循环体内的循环控制语句", item.kind, "把跳出或继续移入循环体")
			}
		case "返回":
			if index != len(items)-1 {
				return functionBlockIssue(items[index+1].name.pos, "返回后没有不可达语句", items[index+1].kind, "移除同一语句列表中的不可达语句")
			}
			kind, problem := constantType(item.value, environment)
			if problem != nil {
				return problem
			}
			if kind != "整数" && kind != "运行时整数" && !(allowResultReturn && kind == "运行时结果") {
				return functionBlockIssue(item.value.pos, "运行时函数循环体内的整数返回", kind, "返回整数或运行时整数表达式")
			}
			if kind == "运行时结果" && !(item.value.kind == "引用" && environment[item.value.text].kind == "运行时结果" || runtimeResultOperandShape(item.value)) {
				return functionBlockIssue(item.value.pos, "可发射的结果返回表达式", kind, "直接返回结果局部、成功/失败调用或调用返回结果的表达式")
			}
		default:
			return functionBlockIssue(item.name.pos, "变量、赋值、若、凡、跳出、继续或返回语句", item.kind, "使用当前运行时循环体支持的语句")
		}
	}
	return nil
}
func functionLocalNames(statements []statement) []string {
	names := []string{}; var visit func([]statement)
	visit = func(items []statement) {
		for _, item := range items {
			if item.kind == "变量" {
				names = append(names, item.name.text)
			}
			visit(item.body); visit(item.otherwise)
		}
	}
	visit(statements)
	return names
}
func copyWritableVariables(source map[string]bool) map[string]bool {
	copy := make(map[string]bool, len(source))
	for name, writable := range source {
		copy[name] = writable
	}
	return copy
}
func validateConsoleProgram(root value, constants map[string]constantValue, statements []statement) *issue {
	variables := map[string]bool{}; environment := copyConstantEnvironment(constants)
	validationEnvironment := copyConstantEnvironment(constants); slotCount := 0
	for _, item := range statements {
		if item.kind != "变量" {
			continue
		}
		if isRuntimeTextValue(item.value) || isRuntimeArgumentText(item.value) || isStoredTextReturnCall(item.value) {
			slotCount += 9 // 64-byte text buffer plus an eight-byte length slot.
			if slotCount > 502 {
				return &issue{code: "S-3040", expected: "不超过 502 个整数等价栈槽", actual: "文本缓冲区、变量与数组元素超过当前原生栈帧容量", suggestion: "缩小固定文本状态或拆分程序状态", pos: item.name.pos}
			}
			continue
		}
		if isRuntimeMapValue(item.value) {
			slotCount += 3
			if slotCount > 502 {
				return &issue{code: "S-3040", expected: "不超过 502 个整数等价栈槽", actual: "映射描述符、数据与变量超过当前原生栈帧容量", suggestion: "缩小映射状态或拆分程序状态", pos: item.name.pos}
			}
			continue
		}
		if isRuntimeResultValue(item.value) {
			slotCount += 2
			if slotCount > 502 {
				return &issue{code: "S-3040", expected: "不超过 502 个整数等价栈槽", actual: "结果标签、值及变量超过当前原生栈帧容量", suggestion: "缩小结果状态或拆分程序状态", pos: item.name.pos}
			}
			continue
		}
		slotCount++
		if item.value.kind == "调用" && item.value.text == "array" {
			slotCount += len(item.value.args) - 1
		}
		if slotCount > 502 {
			return &issue{code: "S-3040", expected: "不超过 502 个整数栈槽", actual: "变量与数组元素超过当前原生栈帧容量", suggestion: "缩小数组或拆分程序状态", pos: item.name.pos}
		}
	}
	var problem *issue; annotateResultStateCallsInStatements(statements, environment)
	var registerResultVariables func([]statement)
	registerResultVariables = func(items []statement) {
		for index := range items {
			if items[index].kind == "变量" {
				if kind, problem := constantType(items[index].value, environment); problem == nil && kind == "运行时结果" {
					environment[items[index].name.text] = constantValue{kind: kind, resultPayload: runtimeResultPayload(items[index].value, environment)}; annotateResultStateCallsInStatements(items[index].body, environment)
				}
			}
			registerResultVariables(items[index].body); registerResultVariables(items[index].otherwise)
		}
	}
	registerResultVariables(statements); environment, problem = validateConsoleStatements(statements, variables, validationEnvironment, true, 0)
	if problem != nil {
		return problem
	}
	for _, item := range statements {
		if item.kind != "变量" {
			continue
		}
		variable, exists := environment[item.name.text]
		if !exists || variable.kind != "运行时结果" {
			continue
		}
		required := 2
		if isTextResultPayload(variable.resultPayload) {
			required = 4
		}
		counted := 1
		if isRuntimeResultValue(item.value) {
			counted = 2
		}
		slotCount += required - counted
		if slotCount > 502 {
			return &issue{code: "S-3040", expected: "不超过 502 个整数等价栈槽", actual: "结果标签、值、文本指针与长度超过当前原生栈帧容量", suggestion: "缩小结果状态或拆分程序状态", pos: item.name.pos}
		}
	}
	seenOutput := false
	for _, item := range root.fields {
		if item.name.text == "能力" {
			continue
		}
		if item.name.text != "输出" {
			return &issue{code: "S-3003", expected: "输出", actual: item.name.text, suggestion: "控制台程序当前只接受一个整数或字符串输出表达式", pos: item.name.pos}
		}
		if seenOutput {
			return &issue{code: "S-3002", expected: "唯一字段", actual: "输出", suggestion: "删除重复输出字段", pos: item.name.pos}
		}
		seenOutput = true; kind, problem := constantType(item.value, environment)
		if problem != nil {
			return problem
		}
		switch kind {
		case "整数":
			if _, problem = integerValue(item.value, environment); problem != nil {
				return problem
			}
		case "布尔值", "运行时布尔值", "运行时整数": // Runtime input and boolean values share the checked integer output path.
		case "字符串":
			if _, problem = evaluateString(item.value, 0, constants); problem != nil {
				return problem
			}
		case "运行时字符串":
			if isRuntimeTextInput(item.value) || isRuntimeFileRead(item.value) || isRuntimeEnvironmentRead(item.value) || isRuntimeEnvironmentNames(item.value) || item.value.kind == "调用" && (item.value.runtimeFunctionText || item.value.text == "argumentText" && len(item.value.args) == 1) {
				break
			}
			if item.value.kind == "调用" && (item.value.text == "substring" || item.value.text == "textConcat" || item.value.text == "arrayGet") {
				break
			}
			if isTextResultValue(item.value, environment) {
				break
			}
			if item.value.kind == "引用" {
				variable, exists := environment[item.value.text]
				if exists && variable.kind == "运行时字符串" {
					break
				}
			}
			return &issue{code: "S-3044", expected: "readText（）、textConcat（）、substring（）或已初始化的运行时文本变量", actual: "运行时文本表达式", suggestion: "使用有界文本函数或先将结果存入文本变量", pos: item.value.pos}
		default:
			return &issue{code: "S-3023", expected: "整数、字符串或布尔输出表达式", actual: kind, suggestion: "输出一个整数、字符串或布尔表达式", pos: item.value.pos}
		}
	}
	if !seenOutput {
		return &issue{code: "S-3006", expected: "输出 字段", actual: "字段缺失", suggestion: "在程序记录中补入 输出 整数表达式", pos: root.pos}
	}
	return nil
}
func validateConsoleCapabilities(root value, statements []statement, constants map[string]constantValue) *issue {
	declared := map[string]bool{}; seenDeclaration := false
	for _, item := range root.fields {
		if item.name.text != "能力" {
			continue
		}
		if seenDeclaration {
			return &issue{code: "S-4104", expected: "唯一能力字段", actual: "重复能力字段", suggestion: "将能力合并到一个字符串数组中", pos: item.name.pos}
		}
		seenDeclaration = true; kind, problem := constantType(item.value, constants)
		if problem != nil {
			return problem
		}
		if kind != "字符串数组" {
			return &issue{code: "S-4105", expected: "编译期字符串能力数组", actual: kind, suggestion: "使用 能力=array（『stdout』，『stdin』）声明能力", pos: item.value.pos}
		}
		items, problem := evaluateArray(item.value, constants)
		if problem != nil {
			return problem
		}
		for _, capability := range items {
			if capability.kind != "字符串" {
				return &issue{code: "S-4105", expected: "编译期字符串能力", actual: capability.kind, suggestion: "能力数组只包含字符串标识", pos: item.value.pos}
			}
			switch capability.text {
			case "stdin", "stdout", "filesystem.read", "filesystem.write", "array.write", "map.write", "args":
			default:
				return &issue{code: "S-4103", expected: "当前支持的 stdout、stdin、filesystem.read、filesystem.write、array.write、map.write 或 args 能力", actual: capability.text, suggestion: "删除未知能力或使用已支持的能力标识", pos: item.value.pos}
			}
			if declared[capability.text] {
				return &issue{code: "S-4104", expected: "互不重复的能力标识", actual: capability.text, suggestion: "每项能力只声明一次", pos: item.value.pos}
			}
			declared[capability.text] = true
		}
	}
	if !seenDeclaration {
		return &issue{code: "S-4101", expected: "能力 字段", actual: "字段缺失", suggestion: "在程序根记录中显式声明 stdout，并按需声明 stdin", pos: root.pos}
	}
	required := map[string]sourcePosition{}
	for _, item := range root.fields {
		if item.name.text == "输出" {
			required["stdout"] = item.name.pos
			if problem := collectInputCapabilities(item.value, required); problem != nil {
				return problem
			}
		}
	}
	stack := append([]statement(nil), statements...)
	for len(stack) > 0 {
		last := len(stack) - 1; item := stack[last]
		stack = stack[:last]; stack = append(stack, item.body...)
		stack = append(stack, item.otherwise...)
		if problem := collectInputCapabilities(item.condition, required); problem != nil {
			return problem
		}
		if problem := collectInputCapabilities(item.value, required); problem != nil {
			return problem
		}
		if item.kind == "删映射键" {
			required["map.write"] = item.name.pos
		}
	}
	for _, capability := range []string{"stdout", "stdin", "filesystem.read", "filesystem.write", "array.write", "map.write", "args"} {
		position, used := required[capability]
		if used && !declared[capability] {
			return &issue{code: "S-4102", expected: capability + " 能力声明", actual: "程序使用了未声明的 " + capability, suggestion: "将该能力加入程序根记录的 能力 字符串数组", pos: position}
		}
	}
	return nil
}
func collectInputCapabilities(input value, required map[string]sourcePosition) *issue {
	walkValue(input, func(current value) bool {
		if current.kind == "调用" {
			capability := ""
			switch current.text {
			case "readInteger", "readText":
				capability = "stdin"
			case "readFile", "readModule", "readModuleResult", "fileSize", "exists", "isDirectory":
				capability = "filesystem.read"
			case "writeFile", "appendFile", "writeByteArray", "removeFile", "renameFile", "replaceFile", "createDirectory", "removeDirectory", "setFileMode", "changeFileMode":
				capability = "filesystem.write"
			case "arrayPush", "arraySet":
				capability = "array.write"
			case "映射置", "映射删":
				capability = "map.write"
			case "argumentCount", "argumentText":
				capability = "args"
			case "readEnvironment", "environmentNames":
				capability = "args"
			}
			if capability != "" {
				if _, exists := required[capability]; !exists {
					required[capability] = current.pos
				}
			}
		}
		return true
	})
	return nil
}
func validateConsoleStatements(statements []statement, variables map[string]bool, environment map[string]constantValue, allowDeclarations bool, loopDepth int) (map[string]constantValue, *issue) {
	for _, item := range statements {
		switch item.kind {
		case "变量":
			if !allowDeclarations {
				return nil, &issue{code: "S-3037", expected: "控制块中的赋值语句", actual: "变量声明", suggestion: "在控制块前声明变量，并在块内使用赋值", pos: item.name.pos}
			}
			if _, exists := environment[item.name.text]; exists {
				return nil, &issue{code: "S-3037", expected: "与常量名不同的唯一变量名", actual: item.name.text, suggestion: "为变量使用一个尚未被常量或变量占用的名称", pos: item.name.pos}
			}
			if _, exists := variables[item.name.text]; exists {
				return nil, &issue{code: "S-3037", expected: "唯一变量名", actual: item.name.text, suggestion: "每个变量只声明一次", pos: item.name.pos}
			}
			if len(variables) >= 502 {
				return nil, &issue{code: "S-3040", expected: "不超过 502 个标量变量", actual: "变量数超过当前原生栈帧容量", suggestion: "拆分程序状态，或等待更大的通用栈帧实现", pos: item.name.pos}
			}
			variables[item.name.text] = true
			if problem := assignConsoleInteger(item, environment, "整数变量初值"); problem != nil {
				return nil, problem
			}
		case "赋值":
			if _, exists := variables[item.name.text]; !exists {
				return nil, &issue{code: "S-3038", expected: "此前声明的可变整数变量", actual: item.name.text, suggestion: "先声明目标变量，再执行赋值", pos: item.name.pos}
			}
			if problem := assignConsoleInteger(item, environment, "整数赋值"); problem != nil {
				return nil, problem
			}
		case "删映射键":
			target := item.value.args[0]
			if target.kind != "引用" || environment[target.text].kind != "整数映射" {
				return nil, &issue{code: "S-3015", expected: "整数映射变量", actual: target.text, suggestion: "先用 映射（） 声明目标映射变量", pos: target.pos}
			}
			keyType, problem := constantType(item.value.args[1], environment)
			if problem != nil {
				return nil, problem
			}
			if keyType != "整数" && keyType != "运行时整数" {
				return nil, &issue{code: "S-3015", expected: "整数映射键", actual: keyType, suggestion: "当前整数映射只接受有符号 64 位键", pos: item.value.args[1].pos}
			}
		case "跳出", "继续":
			if loopDepth == 0 {
				return nil, &issue{code: "S-3067", expected: "凡 循环体内的循环控制语句", actual: item.kind, suggestion: "将跳出或继续语句移入凡循环体", pos: item.name.pos}
			}
		case "若", "凡":
			conditionKind, problem := constantType(item.condition, environment)
			if problem != nil {
				return nil, problem
			}
			if conditionKind != "布尔值" && conditionKind != "运行时布尔值" {
				return nil, &issue{code: "S-3015", expected: "整数比较布尔条件", actual: conditionKind, suggestion: "使用整数比较符或 equals/notEquals 作为条件", pos: item.condition.pos}
			}
			bodyLoopDepth := loopDepth; bodyEnvironment := copyConstantEnvironment(environment)
			if item.kind == "凡" {
				bodyLoopDepth++
				markConsoleVariablesRuntime(assignedConsoleVariables(item.body), bodyEnvironment)
			}
			thenEnvironment, problem := validateConsoleStatements(item.body, variables, bodyEnvironment, false, bodyLoopDepth)
			if problem != nil {
				return nil, problem
			}
			if item.kind == "若" {
				elseEnvironment, elseProblem := validateConsoleStatements(item.otherwise, variables, copyConstantEnvironment(environment), false, loopDepth)
				if elseProblem != nil {
					return nil, elseProblem
				}
				if conditionKind == "布尔值" {
					selected, evalProblem := evaluateBoolean(item.condition, 0, environment)
					if evalProblem != nil {
						return nil, evalProblem
					}
					if selected {
						environment = thenEnvironment
					} else {
						environment = elseEnvironment
					}
				} else {
					markConsoleVariablesRuntime(assignedConsoleVariables(item.body), environment); markConsoleVariablesRuntime(assignedConsoleVariables(item.otherwise), environment)
				}
				continue
			}
			if conditionKind == "布尔值" {
				selected, evalProblem := evaluateBoolean(item.condition, 0, environment)
				if evalProblem != nil {
					return nil, evalProblem
				}
				if !selected {
					continue
				}
			}
			markConsoleVariablesRuntime(assignedConsoleVariables(item.body), environment)
		default:
			return nil, &issue{code: "S-3003", expected: "变量、赋值、若、凡、跳出或继续语句", actual: item.kind, suggestion: "使用已实现的整数变量与控制语句", pos: item.name.pos}
		}
	}
	return environment, nil
}
func assignConsoleInteger(item statement, environment map[string]constantValue, expected string) *issue {
	kind, problem := constantType(item.value, environment)
	if problem != nil {
		return problem
	}
	if kind == "运行时结果" {
		if expected == "整数赋值" && environment[item.name.text].kind != "运行时结果" {
			return &issue{code: "S-3023", expected: "运行时结果变量赋值", actual: kind, suggestion: "将结果值赋给结果变量", pos: item.value.pos}
		}
		payload := environment[item.name.text].resultPayload; incoming := runtimeResultPayload(item.value, environment)
		if incoming != "" && payload != "" && incoming != payload {
			return &issue{code: "S-3023", expected: payload + "载荷结果变量赋值", actual: incoming + "载荷结果", suggestion: "保持结果变量的成功载荷类型不变", pos: item.value.pos}
		}
		if payload == "" {
			payload = incoming
		}
		environment[item.name.text] = constantValue{kind: "运行时结果", resultPayload: payload}
		return nil
	}
	if expected == "整数赋值" && environment[item.name.text].kind == "运行时结果" {
		return &issue{code: "S-3023", expected: "运行时结果变量赋值", actual: kind, suggestion: "使用成功或失败生成结果值", pos: item.value.pos}
	}
	if kind == "运行时字符串" {
		if !isRuntimeTextValue(item.value) && !isRuntimeFileRead(item.value) && !isRuntimeArgumentText(item.value) && !isRuntimeEnvironmentRead(item.value) && !isRuntimeEnvironmentNames(item.value) && !isTextResultValue(item.value, environment) && !isStoredTextReturnCall(item.value) {
			return &issue{code: "S-3044", expected: "有界运行时文本表达式、文件读取、参数文本或文本结果值", actual: "运行时文本表达式", suggestion: "使用 readText（）、readFile（静态路径）、argumentText（下标）、textConcat、有界 substring 或文本结果值初始化或更新文本变量", pos: item.value.pos}
		}
		if expected == "整数赋值" && environment[item.name.text].kind != "运行时字符串" {
			return &issue{code: "S-3023", expected: "整数变量或数组赋值", actual: "运行时字符串", suggestion: "将文本输入赋给文本变量", pos: item.value.pos}
		}
		environment[item.name.text] = constantValue{kind: "运行时字符串"}
		return nil
	}
	if kind == "布尔值" || kind == "运行时布尔值" {
		if expected == "整数赋值" && environment[item.name.text].kind != "运行时布尔值" {
			return &issue{code: "S-3023", expected: "运行时布尔变量赋值", actual: kind, suggestion: "将布尔值赋给布尔变量", pos: item.value.pos}
		}
		environment[item.name.text] = constantValue{kind: "运行时布尔值"}
		return nil
	}
	if expected == "整数赋值" && environment[item.name.text].kind == "运行时布尔值" {
		return &issue{code: "S-3023", expected: "运行时布尔变量赋值", actual: kind, suggestion: "将布尔值赋给布尔变量", pos: item.value.pos}
	}
	if expected == "整数赋值" && environment[item.name.text].kind == "运行时字符串" {
		return &issue{code: "S-3023", expected: "运行时字符串赋值", actual: kind, suggestion: "使用 readText（）更新文本变量", pos: item.value.pos}
	}
	if expected == "整数变量初值" && kind == "整数映射" {
		environment[item.name.text] = constantValue{kind: "整数映射"}
		return nil
	}
	if expected == "整数赋值" && environment[item.name.text].kind == "整数映射" {
		if item.value.kind != "调用" || len(item.value.args) < 2 || item.value.args[0].kind != "引用" || item.value.args[0].text != item.name.text || item.value.text != "映射置" && item.value.text != "映射删" {
			return &issue{code: "S-3023", expected: "对目标映射自身的 映射置（映射，键，值）或 映射删（映射，键）", actual: item.value.text, suggestion: "声明映射变量，并用 赋值 映射=映射置（映射，键，值）或 映射删（映射，键）更新", pos: item.value.pos}
		}
		return nil
	}
	if expected == "整数变量初值" && (kind == "整数数组" || kind == "运行时整数数组" || kind == "布尔值数组" || kind == "运行时布尔值数组") {
		items, issue := runtimeIntegerArrayItems(item.value, environment)
		if issue != nil {
			return issue
		}
		environment[item.name.text] = constantValue{kind: "运行时" + strings.TrimPrefix(strings.TrimSuffix(kind, "数组"), "运行时") + "数组", array: items}
		return nil
	}
	if expected == "整数变量初值" && (kind == "字符串数组" || kind == "运行时字符串数组") {
		items, issue := runtimeTextArrayItems(item.value, environment)
		if issue != nil {
			return issue
		}
		environment[item.name.text] = constantValue{kind: "运行时字符串数组", array: items}
		return nil
	}
	if expected == "整数赋值" && item.value.kind == "调用" && item.value.text == "arrayPush" {
		if len(item.value.args) != 2 || item.value.args[0].kind != "引用" || item.value.args[0].text != item.name.text {
			return arrayIssue(item.value, "S-3033", "对目标变量自身的 arrayPush（数组，值）", "数组目标与赋值目标不一致", "使用 赋值 数组名=arrayPush（数组名，值）")
		}
		array, exists := environment[item.name.text]
		if !exists || !(strings.HasPrefix(array.kind, "运行时") && strings.HasSuffix(array.kind, "数组")) {
			return arrayIssue(item.value, "S-3033", "此前声明的可变数组", item.name.text, "先用 array（…）声明数组变量")
		}
		valueType, valueProblem := constantType(item.value.args[1], environment)
		if valueProblem != nil {
			return valueProblem
		}
		if strings.TrimSuffix(strings.TrimPrefix(array.kind, "运行时"), "数组") != strings.TrimPrefix(valueType, "运行时") {
			return arrayIssue(item.value, "S-3031", "同类数组元素", valueType, "追加与数组元素类型相同的值")
		}
		environment[item.name.text] = array
		return nil
	}
	if expected == "整数赋值" && item.value.kind == "调用" && item.value.text == "arraySet" {
		if len(item.value.args) != 3 || item.value.args[0].kind != "引用" || item.value.args[0].text != item.name.text {
			return arrayIssue(item.value, "S-3033", "对目标变量自身的 arraySet（数组，索引，值）", "数组目标与赋值目标不一致", "使用 赋值 数组名=arraySet（数组名，索引，值）")
		}
		array, exists := environment[item.name.text]
		if !exists || !(strings.HasPrefix(array.kind, "运行时") && strings.HasSuffix(array.kind, "数组")) {
			return arrayIssue(item.value, "S-3033", "此前声明的可变数组", item.name.text, "先用 array（…）声明数组变量")
		}
		indexType, indexProblem := constantType(item.value.args[1], environment)
		if indexProblem != nil {
			return indexProblem
		}
		if indexType != "整数" && indexType != "运行时整数" {
			return arrayIssue(item.value, "S-3034", "整数索引", indexType, "将数组索引改为整数表达式")
		}
		valueType, valueProblem := constantType(item.value.args[2], environment)
		if valueProblem != nil {
			return valueProblem
		}
		if strings.TrimSuffix(strings.TrimPrefix(array.kind, "运行时"), "数组") != strings.TrimPrefix(valueType, "运行时") {
			return arrayIssue(item.value, "S-3031", "同类数组元素", valueType, "写入与数组元素类型相同的值")
		}
		if indexType == "整数" {
			index, indexProblem := integerValue(item.value.args[1], environment)
			if indexProblem != nil {
				return indexProblem
			}
			if index < 0 {
				return arrayIssue(item.value, "S-3032", "非负数组索引", strconv.FormatInt(index, 10), "将数组索引调整为非负数")
			}
		}
		itemKind := "运行时" + strings.TrimSuffix(strings.TrimPrefix(array.kind, "运行时"), "数组")
		for index := range array.array {
			array.array[index] = constantValue{kind: itemKind}
		}
		environment[item.name.text] = array
		return nil
	}
	if kind != "整数" && kind != "运行时整数" {
		return &issue{code: "S-3023", expected: expected, actual: kind, suggestion: "使用整数表达式初始化或更新变量", pos: item.value.pos}
	}
	if kind == "运行时整数" {
		environment[item.name.text] = constantValue{kind: "运行时整数"}
		return nil
	}
	integer, problem := integerValue(item.value, environment)
	if problem != nil {
		return problem
	}
	environment[item.name.text] = constantValue{kind: "整数", integer: integer}
	return nil
}
func isRuntimeResultValue(input value) bool {
	if input.kind != "调用" {
		return false
	}
	if input.runtimeFunction && input.runtimeFunctionResult {
		return true
	}
	if input.text == "成功" || input.text == "失败" || isModuleReadResult(input) {
		return true
	}
	return input.text == "ifElse" && len(input.args) == 3 && isRuntimeResultValue(input.args[1]) && isRuntimeResultValue(input.args[2])
}
func runtimeResultPayload(input value, environment map[string]constantValue) string {
	if isModuleReadResult(input) {
		return "运行时文本"
	}
	if input.runtimeFunction && input.runtimeFunctionResult {
		if input.runtimeFunctionPayload != "" {
			return input.runtimeFunctionPayload
		}
		return "整数"
	}
	if input.kind == "引用" {
		if environment[input.text].kind == "运行时结果" && environment[input.text].resultPayload == "" {
			return "整数"
		}
		return environment[input.text].resultPayload
	}
	if input.kind == "调用" && input.text == "ifElse" && len(input.args) == 3 {
		whenTrue := runtimeResultPayload(input.args[1], environment); whenFalse := runtimeResultPayload(input.args[2], environment)
		if whenTrue == "" {
			return whenFalse
		}
		if whenFalse == "" || whenTrue == whenFalse {
			return whenTrue
		}
		if isTextResultPayload(whenTrue) || isTextResultPayload(whenFalse) {
			return "文本"
		}
		return ""
	}
	if input.kind != "调用" || input.text != "成功" || len(input.args) != 1 {
		return ""
	}
	if isRuntimeTextValue(input.args[0]) || isRuntimeArgumentText(input.args[0]) {
		return "运行时文本"
	}
	kind, problem := constantType(input.args[0], environment)
	if problem != nil {
		return ""
	}
	if kind == "字符串" {
		return "文本"
	}
	if kind == "整数" || kind == "运行时整数" {
		return "整数"
	}
	return ""
}
func isTextResultPayload(payload string) bool {
	return payload == "文本" || payload == "运行时文本"
}
func isRuntimeMapValue(input value) bool {
	return input.kind == "调用" && input.text == "映射" && len(input.args) == 0
}
func runtimeIntegerArrayItems(input value, environment map[string]constantValue) ([]constantValue, *issue) {
	if input.kind != "调用" || input.text != "array" || len(input.args) == 0 {
		return nil, arrayIssue(input, "S-3033", "非空标量数组构造", input.kind, "使用 array（同类型整数或布尔元素，…）初始化可变标量数组")
	}
	items := make([]constantValue, 0, len(input.args))
	for _, argument := range input.args {
		kind, problem := constantType(argument, environment)
		if problem != nil {
			return nil, problem
		}
		if kind != "整数" && kind != "运行时整数" && kind != "布尔值" && kind != "运行时布尔值" {
			return nil, arrayIssue(input, "S-3031", "整数或布尔数组元素", kind, "可变数组元素必须保持同一标量类型")
		}
		item := constantValue{kind: kind}
		if kind == "整数" {
			item.integer, problem = integerValue(argument, environment)
			if problem != nil {
				return nil, problem
			}
		}
		items = append(items, item)
	}
	return items, nil
}
func runtimeTextArrayItems(input value, environment map[string]constantValue) ([]constantValue, *issue) {
	if input.kind != "调用" || input.text != "array" || len(input.args) == 0 {
		return nil, arrayIssue(input, "S-3033", "非空字符串数组构造", input.kind, "使用 array（文本元素，…）初始化可变数组")
	}
	items := make([]constantValue, 0, len(input.args))
	for _, argument := range input.args {
		kind, problem := constantType(argument, environment)
		if problem != nil {
			return nil, problem
		}
		if kind != "字符串" && kind != "运行时字符串" {
			return nil, arrayIssue(input, "S-3031", "字符串数组元素", kind, "可变字符串数组只接受文本值")
		}
		item := constantValue{kind: kind}
		if kind == "字符串" {
			item.text, problem = evaluateString(argument, 0, environment)
			if problem != nil {
				return nil, problem
			}
		}
		items = append(items, item)
	}
	return items, nil
}
func assignedConsoleVariables(statements []statement) map[string]bool {
	assigned := map[string]bool{}
	for _, item := range statements {
		if item.kind == "赋值" {
			assigned[item.name.text] = true
		}
		for name := range assignedConsoleVariables(item.body) {
			assigned[name] = true
		}
		for name := range assignedConsoleVariables(item.otherwise) {
			assigned[name] = true
		}
	}
	return assigned
}
func markConsoleVariablesRuntime(names map[string]bool, environment map[string]constantValue) {
	for name := range names {
		if variable, exists := environment[name]; exists && (variable.kind == "运行时字符串" || variable.kind == "运行时布尔值" || variable.kind == "运行时结果") {
			environment[name] = variable
		} else if exists && variable.kind == "整数映射" {
			environment[name] = variable
		} else if exists && strings.HasPrefix(variable.kind, "运行时") && strings.HasSuffix(variable.kind, "数组") {
			itemKind := "运行时" + strings.TrimSuffix(strings.TrimPrefix(variable.kind, "运行时"), "数组")
			for index := range variable.array {
				variable.array[index] = constantValue{kind: itemKind}
			}
			environment[name] = variable
		} else {
			environment[name] = constantValue{kind: "运行时整数"}
		}
	}
}
func copyConstantEnvironment(constants map[string]constantValue) map[string]constantValue {
	environment := make(map[string]constantValue, len(constants))
	for name, constant := range constants {
		constant.array = append([]constantValue(nil), constant.array...); environment[name] = constant
	}
	return environment
}
func stringValue(input value, constants map[string]constantValue) (string, bool) {
	text, problem := evaluateString(input, 0, constants)
	return text, problem == nil
}
func isResultOperand(input value) bool {
	if input.kind == "引用" {
		return true
	}
	if input.kind != "调用" {
		return false
	}
	if input.text == "成功" || input.text == "失败" {
		return true
	}
	return input.runtimeFunction && input.runtimeFunctionResult || isModuleReadResult(input) ||
		input.text == "ifElse" && len(input.args) == 3 &&
			(isResultOperand(input.args[1]) || isResultOperand(input.args[2]))
}
func validateFilePathArgument(input value, constants map[string]constantValue, code, expected, suggestion string) *issue {
	pathType, problem := constantType(input, constants)
	if problem != nil {
		return problem
	}
	if pathType != "字符串" && !isRuntimeArgumentText(input) {
		return &issue{code: code, expected: expected, actual: pathType, suggestion: suggestion, pos: input.pos}
	}
	if pathType == "字符串" {
		path, problem := evaluateString(input, 0, constants)
		if problem != nil {
			return problem
		}
		if path == "" || strings.IndexByte(path, 0) >= 0 {
			return &issue{code: code, expected: "非空且不含 NUL 的文件路径", actual: path, suggestion: "使用进程当前目录可解析的普通文本路径", pos: input.pos}
		}
	}
	return nil
}
func validateEnvironmentNameArgument(input value, constants map[string]constantValue) *issue {
	if input.kind == "调用" && input.text == "argumentText" && len(input.args) == 1 { return &issue{code: "S-3083", expected: "编译期文本名称", actual: "运行时字符串", suggestion: "当前环境变量查询仅接受非空静态文本名称", pos: input.pos} }
	nameType, problem := constantType(input, constants)
	if problem != nil {
		return problem
	}
	if nameType != "字符串" {
		return &issue{code: "S-3083", expected: "编译期文本名称", actual: nameType, suggestion: "当前环境变量查询仅接受非空静态文本名称", pos: input.pos}
	}
	name, problem := evaluateString(input, 0, constants)
	if problem != nil {
		return problem
	}
	if name == "" || strings.IndexByte(name, 0) >= 0 || strings.IndexByte(name, '=') >= 0 {
		return &issue{code: "S-3083", expected: "非空且不含 NUL 与等号的文本名称", actual: name, suggestion: "使用普通环境变量名称", pos: input.pos}
	}
	return nil
}
func constantType(input value, constants map[string]constantValue) (string, *issue) {
	type typeFrame struct {
		current value; visited bool
	}
	frames := []typeFrame{{current: input}}; types := []string{}
	for len(frames) > 0 {
		index := len(frames) - 1; frame := frames[index]
		frames = frames[:index]; current := frame.current
		switch current.kind {
		case "字符串":
			types = append(types, "字符串")
		case "整数":
			if _, err := strconv.ParseInt(current.text, 10, 64); err != nil {
				return "", &issue{code: "S-3004", expected: "范围内的有符号 64 位整数", actual: current.text, suggestion: "将整数缩小到 -9223372036854775808 至 9223372036854775807", pos: current.pos}
			}
			types = append(types, "整数")
		case "引用":
			constant, exists := constants[current.text]
			if !exists {
				return "", unresolvedConstant(current)
			}
			types = append(types, constant.kind)
		case "表达式":
			if current.left == nil || (current.operator != "negate" && current.right == nil) {
				return "", &issue{code: "S-3023", expected: "有效的字符串或整数表达式", actual: current.operator, suggestion: "补齐当前运算符的操作数", pos: current.pos}
			}
			if !frame.visited {
				frames = append(frames, typeFrame{current: current, visited: true})
				if current.right != nil {
					frames = append(frames, typeFrame{current: *current.right})
				}
				frames = append(frames, typeFrame{current: *current.left})
				continue
			}
			if current.operator == "negate" {
				operand := types[len(types)-1]; types = types[:len(types)-1]
				if operand == "运行时整数" {
					types = append(types, "运行时整数")
					continue
				}
				if operand != "整数" {
					return "", &issue{code: "S-3023", expected: "整数一元负号", actual: operand, suggestion: "只对整数常量使用一元负号", pos: current.pos}
				}
				types = append(types, "整数")
				continue
			}
			right := types[len(types)-1]; left := types[len(types)-2]
			types = types[:len(types)-2]
			if left == "运行时整数" || right == "运行时整数" {
				if (left != "整数" && left != "运行时整数") || (right != "整数" && right != "运行时整数") {
					return "", &issue{code: "S-3023", expected: "整数运行时算术操作数", actual: left + " " + current.operator + " " + right, suggestion: "整数运算符两侧使用整数值", pos: current.pos}
				}
				if current.operator != "+" && current.operator != "-" && current.operator != "*" && current.operator != "/" && current.operator != "%" && current.operator != "<<" && current.operator != ">>" {
					return "", &issue{code: "S-3023", expected: "受支持的整数运行时运算", actual: current.operator, suggestion: "使用加减乘除、余数或移位运算符", pos: current.pos}
				}
				types = append(types, "运行时整数")
				continue
			}
			if left != right {
				return "", &issue{code: "S-3023", expected: "同类型的表达式操作数", actual: left + " " + current.operator + " " + right, suggestion: "让运算符两侧使用相同类型；减、乘、除只接受整数", pos: current.pos}
			}
			if current.operator == "+" && (left == "字符串" || left == "整数") {
				types = append(types, left)
				continue
			}
			if (current.operator == "-" || current.operator == "*" || current.operator == "/" || current.operator == "%" || current.operator == "<<" || current.operator == ">>") && left == "整数" {
				types = append(types, "整数")
				continue
			}
			return "", &issue{code: "S-3023", expected: "字符串拼接 + 或整数算术/移位运算", actual: left + " " + current.operator + " " + right, suggestion: "使用支持的运算符和对应类型", pos: current.pos}
		case "比较":
			if current.left == nil || current.right == nil {
				return "", &issue{code: "S-3023", expected: "完整的整数比较表达式", actual: current.operator, suggestion: "在比较符两侧提供整数表达式", pos: current.pos}
			}
			left, problem := constantType(*current.left, constants)
			if problem != nil {
				return "", problem
			}
			right, problem := constantType(*current.right, constants)
			if problem != nil {
				return "", problem
			}
			leftText := left == "字符串" || left == "运行时字符串"; rightText := right == "字符串" || right == "运行时字符串"
			if leftText && rightText &&
				(current.operator == "<" || current.operator == "<=" || current.operator == ">" || current.operator == ">=") {
				if left == "运行时字符串" && !runtimeTextComparisonOperand(*current.left) || right == "运行时字符串" && !runtimeTextComparisonOperand(*current.right) {
					return "", &issue{code: "S-3023", expected: "受支持的运行时文本操作数或编译期字符串", actual: left + " " + current.operator + " " + right, suggestion: "使用文本变量、readText、textConcat、substring 或文本数组元素", pos: current.pos}
				}
				if left == "运行时字符串" || right == "运行时字符串" {
					types = append(types, "运行时布尔值")
				} else {
					types = append(types, "布尔值")
				}
				continue
			}
			if (left != "整数" && left != "运行时整数") || (right != "整数" && right != "运行时整数") {
				return "", &issue{code: "S-3023", expected: "两个整数比较操作数", actual: left + " " + current.operator + " " + right, suggestion: "整数比较符两侧均使用整数表达式", pos: current.pos}
			}
			if left == "运行时整数" || right == "运行时整数" {
				types = append(types, "运行时布尔值")
			} else {
				types = append(types, "布尔值")
			}
		case "逻辑":
			if current.left == nil || current.right == nil || current.operator != "且" && current.operator != "故" {
				return "", &issue{code: "S-3015", expected: "完整的且/故布尔表达式", actual: current.operator, suggestion: "在且、故两侧提供布尔条件", pos: current.pos}
			}
			left, problem := constantType(*current.left, constants)
			if problem != nil {
				return "", problem
			}
			right, problem := constantType(*current.right, constants)
			if problem != nil {
				return "", problem
			}
			if (left != "布尔值" && left != "运行时布尔值") || (right != "布尔值" && right != "运行时布尔值") {
				return "", &issue{code: "S-3015", expected: "两个布尔比较条件", actual: left + " " + current.operator + " " + right, suggestion: "且、故两侧都使用比较或相等条件", pos: current.pos}
			}
			if left == "运行时布尔值" || right == "运行时布尔值" {
				types = append(types, "运行时布尔值")
			} else {
				types = append(types, "布尔值")
			}
		case "调用":
			if current.runtimeFunction {
				argumentTypes := current.runtimeArgumentTypes
				if len(argumentTypes) != len(current.args) {
					return "", &issue{code: "S-3062", expected: "运行时函数实参类型与参数一致", actual: strconv.Itoa(len(argumentTypes)) + " 个登记类型，" + strconv.Itoa(len(current.args)) + " 个实参", suggestion: "重新编译；若仍失败，检查运行时函数调用注解", pos: current.pos}
				}
				for index, argument := range current.args {
					argumentType, problem := constantType(argument, constants)
					if problem != nil {
						return "", problem
					}
					switch argumentTypes[index] {
					case "运行时字符串":
						if argumentType != "字符串" && argumentType != "运行时字符串" {
							return "", &issue{code: "S-3062", expected: "运行时文本参数实参", actual: argumentType, suggestion: "传入编译期字符串或受支持的运行时文本表达式", pos: argument.pos}
						}
					case "运行时结果":
						if argumentType != "运行时结果" {
							return "", &issue{code: "S-3062", expected: "结果变量引用", actual: argumentType, suggestion: "先保存结果值再传给运行时函数", pos: argument.pos}
						}
						payload := runtimeResultPayload(argument, constants)
						if argument.kind == "引用" {
							if stored := constants[argument.text].resultPayload; stored != "" {
								payload = stored
							}
						}
						expectedText := index < len(current.runtimeArgumentPayloads) && isTextResultPayload(current.runtimeArgumentPayloads[index])
						if isTextResultPayload(payload) && !expectedText {
							return "", &issue{code: "S-3023", expected: "整数载荷结果参数", actual: "运行时文本载荷结果", suggestion: "先把文本结果载荷复制到文本变量，再传递整数状态", pos: argument.pos}
						}
						if expectedText && payload != "" && !isTextResultPayload(payload) {
							return "", &issue{code: "S-3062", expected: "文本载荷结果参数", actual: payload + "载荷结果", suggestion: "保持该结果形参的成功载荷为文本", pos: argument.pos}
						}
					case "运行时布尔值":
						if argumentType != "布尔值" && argumentType != "运行时布尔值" {
							return "", &issue{code: "S-3062", expected: "运行时布尔条件参数", actual: argumentType, suggestion: "使用比较或逻辑条件作为 ifElse 条件", pos: argument.pos}
						}
					case "运行时整数":
						if argumentType == "运行时结果" {
							payload := runtimeResultPayload(argument, constants)
							if stored := constants[argument.text].resultPayload; stored != "" {
								payload = stored
							}
							if isTextResultPayload(payload) {
								return "", &issue{code: "S-3023", expected: "整数载荷结果参数", actual: "运行时文本载荷结果", suggestion: "先把文本结果载荷复制到文本变量，再传递整数状态", pos: argument.pos}
							}
							continue
						}
						if argumentType != "整数" && argumentType != "运行时整数" {
							return "", &issue{code: "S-3062", expected: "运行时整数参数实参", actual: argumentType, suggestion: "运行时函数整数参数只接受有符号 64 位整数表达式", pos: argument.pos}
						}
					default:
						return "", &issue{code: "S-3062", expected: "已知运行时参数类型", actual: argumentTypes[index], suggestion: "检查运行时函数参数类型推断", pos: argument.pos}
					}
				}
				if current.runtimeFunctionResult {
					types = append(types, "运行时结果")
				} else if current.runtimeFunctionText {
					types = append(types, "运行时字符串")
				} else {
					types = append(types, "运行时整数")
				}
				continue
			}
			kind := ""
			switch current.text {
			case "成功", "失败":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3011", expected: "1 个成功载荷或错误码参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "为成功结果提供整数或编译期文本，为失败提供错误码", pos: current.pos}
				}
				argumentType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if current.text == "成功" {
					if argumentType == "运行时字符串" && (isRuntimeTextValue(current.args[0]) || isRuntimeArgumentText(current.args[0]) || current.args[0].kind == "调用" && current.args[0].text == "arrayGet" && len(current.args[0].args) == 2 && current.args[0].args[0].kind == "引用") {
					} else if argumentType != "整数" && argumentType != "运行时整数" && argumentType != "字符串" {
						return "", &issue{code: "S-3023", expected: "整数、编译期文本或有界运行时文本成功值", actual: argumentType, suggestion: "成功结果支持整数、编译期文本、readText（）、textConcat（）或 substring（）载荷", pos: current.args[0].pos}
					}
				} else {
					if argumentType != "整数" && argumentType != "运行时整数" {
						return "", &issue{code: "S-3023", expected: "整数错误码", actual: argumentType, suggestion: "失败结果携带 1 至 255 的整数错误码", pos: current.args[0].pos}
					}
					if argumentType == "整数" {
						code, problem := integerValue(current.args[0], constants)
						if problem != nil {
							return "", problem
						}
						if code < 1 || code > 255 {
							return "", &issue{code: "S-3013", expected: "1 至 255 的错误码", actual: strconv.FormatInt(code, 10), suggestion: "使用可观察的进程退出码范围", pos: current.args[0].pos}
						}
					}
				}
				kind = "运行时结果"
			case "是否失败":
				if len(current.args) != 1 || !isResultOperand(current.args[0]) {
					return "", &issue{code: "S-3013", expected: "1 个结果变量或返回结果的调用", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "查询结果变量，或直接查询返回结果的调用", pos: current.pos}
				}
				kind = "运行时布尔值"; argumentType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if argumentType != "运行时结果" {
					return "", &issue{code: "S-3015", expected: "结果变量", actual: argumentType, suggestion: "使用成功或失败生成的结果值", pos: current.args[0].pos}
				}
			case "结果值", "错误码":
				if len(current.args) != 1 || !isResultOperand(current.args[0]) {
					return "", &issue{code: "S-3013", expected: "1 个结果变量或返回结果的调用", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "读取结果变量，或直接读取返回结果的调用", pos: current.pos}
				}
				argumentType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if argumentType != "运行时结果" {
					return "", &issue{code: "S-3015", expected: "结果变量", actual: argumentType, suggestion: "使用成功或失败生成的结果值", pos: current.args[0].pos}
				}
				if current.text == "错误码" {
					kind = "运行时整数"
				} else if isTextResultPayload(runtimeResultPayload(current.args[0], constants)) {
					kind = "运行时字符串"
				} else {
					kind = "运行时整数"
				}
			case "映射":
				if len(current.args) != 0 {
					return "", &issue{code: "S-3011", expected: "0 个参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "使用 映射（）初始化空映射", pos: current.pos}
				}
				kind = "整数映射"
			case "映射删":
				if len(current.args) != 2 || current.args[0].kind != "引用" {
					return "", &issue{code: "S-3011", expected: "映射变量和整数键", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "使用 赋值 映射=映射删（映射，整数键）删除", pos: current.pos}
				}
				mapType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if mapType != "整数映射" {
					return "", &issue{code: "S-3015", expected: "整数映射变量", actual: mapType, suggestion: "先用 映射（）声明映射变量", pos: current.args[0].pos}
				}
				keyType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if keyType != "整数" && keyType != "运行时整数" {
					return "", &issue{code: "S-3015", expected: "整数映射键", actual: keyType, suggestion: "当前整数映射只接受有符号 64 位键", pos: current.args[1].pos}
				}
				kind = "整数映射"
			case "映射置":
				if len(current.args) != 3 || current.args[0].kind != "引用" {
					return "", &issue{code: "S-3011", expected: "映射变量、整数键和整数值", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "使用 赋值 映射=映射置（映射，键，值）更新映射", pos: current.pos}
				}
				mapType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if mapType != "整数映射" {
					return "", &issue{code: "S-3015", expected: "整数映射变量", actual: mapType, suggestion: "先用 映射（）声明映射变量", pos: current.args[0].pos}
				}
				for _, argument := range current.args[1:] {
					argumentType, problem := constantType(argument, constants)
					if problem != nil {
						return "", problem
					}
					if argumentType != "整数" && argumentType != "运行时整数" {
						return "", &issue{code: "S-3015", expected: "整数映射键和值", actual: argumentType, suggestion: "当前整数映射只接受有符号 64 位键和值", pos: argument.pos}
					}
				}
				kind = "整数映射"
			case "映射取":
				if len(current.args) != 3 || current.args[0].kind != "引用" {
					return "", &issue{code: "S-3011", expected: "映射变量、整数键和整数默认值", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "使用 映射取（映射，键，默认值）读取", pos: current.pos}
				}
				mapType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if mapType != "整数映射" {
					return "", &issue{code: "S-3015", expected: "整数映射变量", actual: mapType, suggestion: "使用映射变量读取", pos: current.args[0].pos}
				}
				for _, argument := range current.args[1:] {
					argumentType, problem := constantType(argument, constants)
					if problem != nil {
						return "", problem
					}
					if argumentType != "整数" && argumentType != "运行时整数" {
						return "", &issue{code: "S-3015", expected: "整数映射键和默认值", actual: argumentType, suggestion: "当前整数映射只接受有符号 64 位键和值", pos: argument.pos}
					}
				}
				kind = "运行时整数"
			case "映射有":
				if len(current.args) != 2 || current.args[0].kind != "引用" {
					return "", &issue{code: "S-3011", expected: "映射变量和整数键", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "使用 映射有（映射，键）", pos: current.pos}
				}
				mapType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if mapType != "整数映射" {
					return "", &issue{code: "S-3015", expected: "整数映射变量", actual: mapType, suggestion: "使用映射变量查询", pos: current.args[0].pos}
				}
				keyType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if keyType != "整数" && keyType != "运行时整数" {
					return "", &issue{code: "S-3015", expected: "整数映射键", actual: keyType, suggestion: "当前整数映射只接受有符号 64 位键", pos: current.args[1].pos}
				}
				kind = "运行时布尔值"
			case "映射长":
				if len(current.args) != 1 || current.args[0].kind != "引用" {
					return "", &issue{code: "S-3011", expected: "1 个映射变量引用", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "使用 映射长（映射）", pos: current.pos}
				}
				mapType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if mapType != "整数映射" {
					return "", &issue{code: "S-3015", expected: "整数映射变量", actual: mapType, suggestion: "使用映射变量读取长度", pos: current.args[0].pos}
				}
				kind = "运行时整数"
			case "substring":
				resolvedKind, problem := runtimeTextSubstringType(current, constants)
				if problem != nil {
					return "", problem
				}
				kind = resolvedKind
			case "equals", "notEquals":
				resolvedKind, problem := booleanCallType(current, constants)
				if problem != nil {
					return "", problem
				}
				kind = resolvedKind
			case "ifElse":
				resolvedKind, problem := ifElseType(current, constants)
				if problem != nil {
					return "", problem
				}
				kind = resolvedKind
			case "bitAnd", "bitOr", "bitXor":
				if len(current.args) != 2 {
					return "", &issue{code: "S-3021", expected: "2 个位运算参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供两个有符号 64 位整数表达式", pos: current.pos}
				}
				leftType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				rightType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if leftType != "整数" && leftType != "运行时整数" || rightType != "整数" && rightType != "运行时整数" {
					return "", &issue{code: "S-3021", expected: "2 个整数位运算参数", actual: leftType + " 与 " + rightType, suggestion: "向位函数提供两个整数表达式", pos: current.pos}
				}
				kind = "整数"
				if leftType == "运行时整数" || rightType == "运行时整数" {
					kind = "运行时整数"
				}
			case "readInteger":
				if len(current.args) != 0 {
					return "", &issue{code: "S-3043", expected: "readInteger（）无参数调用", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "删除 readInteger 的参数", pos: current.pos}
				}
				kind = "运行时整数"
			case "readText":
				if len(current.args) != 0 {
					return "", &issue{code: "S-3044", expected: "readText（）无参数调用", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "删除 readText 的参数", pos: current.pos}
				}
				kind = "运行时字符串"
			case "readFile":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3070", expected: "readFile（静态文本或参数文本路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个不含 NUL 的编译期路径或 argumentText（下标）", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3070", "编译期文本或 argumentText（下标）路径", "当前文件读取接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				kind = "运行时字符串"
			case "readModule", "readModuleResult":
				if problem := validateModuleRead(current, constants); problem != nil {
					return "", problem
				}
				kind = "运行时字符串"
				if isModuleReadResult(current) {
					kind = "运行时结果"
				}
			case "writeFile", "appendFile", "writeByteArray":
				if len(current.args) != 2 {
					return "", &issue{code: "S-3071", expected: current.text + "（静态或参数文本路径，内容）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供静态或参数文本路径和要写入的内容", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3071", "编译期文本或 argumentText（下标）路径", "当前文件写入接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				textType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if current.text == "writeByteArray" {
					if textType != "运行时整数数组" || current.args[1].kind != "引用" {
						return "", &issue{code: "S-3074", expected: "运行时整数数组变量", actual: textType, suggestion: "将 0 至 255 的字节保存在整数数组变量中", pos: current.args[1].pos}
					}
					types = append(types, "运行时整数")
					continue
				}
				if textType != "字符串" && textType != "运行时字符串" {
					return "", &issue{code: "S-3071", expected: "文本写入内容", actual: textType, suggestion: "第二个参数使用静态文本或有界运行时文本表达式", pos: current.args[1].pos}
				}
				kind = "运行时整数"
			case "fileSize":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3080", expected: "fileSize（静态文本或参数文本路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个已存在文件的路径", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3080", "编译期文本或 argumentText（下标）路径", "当前文件元数据查询接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				kind = "运行时整数"
			case "exists":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3081", expected: "exists（静态文本或参数文本路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个待查询的文件或目录路径", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3081", "编译期文本或 argumentText（下标）路径", "当前存在性查询接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				kind = "运行时整数"
			case "isDirectory":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3081", expected: "isDirectory（静态文本或参数文本路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个待查询的文件或目录路径", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3081", "编译期文本或 argumentText（下标）路径", "当前目录类型查询接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				kind = "运行时整数"
			case "readEnvironment":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3083", expected: "readEnvironment（静态文本名称）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个非空环境变量名称", pos: current.pos}
				}
				if problem := validateEnvironmentNameArgument(current.args[0], constants); problem != nil {
					return "", problem
				}
				kind = "运行时字符串"
			case "environmentNames":
				if len(current.args) != 0 {
					return "", &issue{code: "S-3083", expected: "environmentNames（）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "不提供参数，返回按字节顺序排序并逐行输出的环境变量名称文本", pos: current.pos}
				}
				kind = "运行时字符串"
			case "removeFile":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3075", expected: "removeFile（静态或参数文本路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供要删除的非空文件路径", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3075", "编译期文本或 argumentText（下标）路径", "当前文件删除接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				kind = "运行时整数"
			case "renameFile":
				if len(current.args) != 2 {
					return "", &issue{code: "S-3076", expected: "renameFile（源路径，目标路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供非空的源路径和目标路径", pos: current.pos}
				}
				for _, argument := range current.args {
					if problem := validateFilePathArgument(argument, constants, "S-3076", "编译期文本或 argumentText（下标）路径", "文件重命名接受静态路径或命令行参数文本"); problem != nil {
						return "", problem
					}
				}
				kind = "运行时整数"
			case "replaceFile":
				if len(current.args) != 2 { return "", &issue{code: "S-3082", expected: "replaceFile（源路径，目标路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供非空的源路径和目标路径", pos: current.pos} }
				for _, argument := range current.args { if problem := validateFilePathArgument(argument, constants, "S-3082", "编译期文本或 argumentText（下标）路径", "文件原子替换接受静态路径或命令行参数文本"); problem != nil { return "", problem } }
				kind = "运行时整数"
			case "createDirectory", "removeDirectory":
				if len(current.args) != 1 {
					return "", &issue{code: map[bool]string{true: "S-3077", false: "S-3078"}[current.text == "createDirectory"], expected: current.text + "（静态或参数文本路径）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供非空目录路径", pos: current.pos}
				}
				code := map[bool]string{true: "S-3077", false: "S-3078"}[current.text == "createDirectory"]
				if problem := validateFilePathArgument(current.args[0], constants, code, "编译期文本或 argumentText（下标）路径", "当前目录操作接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				kind = "运行时整数"
			case "setFileMode", "changeFileMode":
				if len(current.args) != 2 {
					return "", &issue{code: "S-3079", expected: "setFileMode（静态或参数文本路径，八进制权限）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供路径和 0 至 511 的权限整数", pos: current.pos}
				}
				if problem := validateFilePathArgument(current.args[0], constants, "S-3079", "编译期文本或 argumentText（下标）路径", "当前权限设置接受静态路径或命令行参数文本"); problem != nil {
					return "", problem
				}
				modeType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if modeType != "整数" && modeType != "运行时整数" {
					return "", &issue{code: "S-3079", expected: "八进制权限整数", actual: modeType, suggestion: "使用 0 至 511 的整数权限", pos: current.args[1].pos}
				}
				if modeType == "整数" {
					mode, problem := integerValue(current.args[1], constants)
					if problem != nil {
						return "", problem
					}
					if mode < 0 || mode > 511 {
						return "", &issue{code: "S-3079", expected: "0 至 511 的权限整数", actual: strconv.FormatInt(mode, 10), suggestion: "使用 0 至 0777 的权限位", pos: current.args[1].pos}
					}
				}
				kind = "运行时整数"
			case "argumentCount":
				if len(current.args) != 0 {
					return "", &issue{code: "S-3069", expected: "argumentCount（）无参数调用", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "删除 argumentCount 的参数", pos: current.pos}
				}
				kind = "运行时整数"
			case "argumentText":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3069", expected: "argumentText（整数下标）", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个非负整数下标表达式", pos: current.pos}
				}
				indexType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if indexType != "整数" && indexType != "运行时整数" {
					return "", &issue{code: "S-3069", expected: "整数参数下标", actual: indexType, suggestion: "使用整数常量、变量或整数表达式读取参数", pos: current.args[0].pos}
				}
				if indexType == "整数" {
					index, problem := integerValue(current.args[0], constants)
					if problem != nil {
						return "", problem
					}
					if index < 0 {
						return "", &issue{code: "S-3069", expected: "非负参数下标", actual: strconv.FormatInt(index, 10), suggestion: "命令行参数从 0 开始编号", pos: current.args[0].pos}
					}
				}
				kind = "运行时字符串"
			case "textConcat":
				resolvedKind, problem := runtimeTextConcatType(current, constants)
				if problem != nil {
					return "", problem
				}
				kind = resolvedKind
			case "textByteLength", "textByteAt":
				resolvedKind, problem := textByteType(current, constants)
				if problem != nil {
					return "", problem
				}
				kind = resolvedKind
			case "textLength":
				if len(current.args) != 1 {
					return "", &issue{code: "S-3044", expected: "1 个文本长度参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个字符串或已初始化的文本变量", pos: current.pos}
				}
				textKind, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if textKind == "字符串" {
					kind = "整数"
				} else if textKind == "运行时字符串" && (current.args[0].kind == "引用" || isRuntimeTextValue(current.args[0]) || isRuntimeArgumentText(current.args[0]) || current.args[0].kind == "调用" && current.args[0].text == "readEnvironment" || current.args[0].kind == "调用" && current.args[0].text == "arrayGet" || isTextResultValue(current.args[0], constants)) {
					kind = "运行时整数"
				} else {
					return "", &issue{code: "S-3044", expected: "编译期字符串或受支持的运行时文本值", actual: textKind, suggestion: "使用文本变量、readText（）、textConcat（）或 substring（）结果", pos: current.args[0].pos}
				}
			case "array":
				if len(current.args) == 0 {
					return "", arrayIssue(current, "S-3030", "至少一个同类型字符串或整数元素", "空数组", "补入一个或多个同类型元素")
				}
				elementType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				baseType := strings.TrimPrefix(elementType, "运行时")
				if baseType != "字符串" && baseType != "整数" && baseType != "布尔值" {
					return "", arrayIssue(current, "S-3031", "一维字符串、整数或布尔值数组", "嵌套数组", "将元素改为同类型标量")
				}
				runtimeElements := strings.HasPrefix(elementType, "运行时")
				for _, argument := range current.args[1:] {
					argumentType, argumentProblem := constantType(argument, constants)
					if argumentProblem != nil {
						return "", argumentProblem
					}
					argumentBase := strings.TrimPrefix(argumentType, "运行时")
					if argumentBase != baseType {
						return "", arrayIssue(current, "S-3031", baseType+" 数组元素", argumentType, "数组中所有元素必须为同一类型")
					}
					runtimeElements = runtimeElements || strings.HasPrefix(argumentType, "运行时")
				}
				kind = baseType + "数组"
				if runtimeElements {
					kind = "运行时" + baseType + "数组"
				}
			case "arrayGet":
				if len(current.args) != 2 {
					return "", arrayIssue(current, "S-3035", "2 个参数", strconv.Itoa(len(current.args))+" 个参数", "提供数组和零起始整数索引")
				}
				arrayType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if arrayType != "字符串数组" && arrayType != "整数数组" && arrayType != "布尔值数组" && arrayType != "运行时整数数组" && arrayType != "运行时字符串数组" && arrayType != "运行时布尔值数组" {
					return "", arrayIssue(current, "S-3033", "数组", arrayType, "将第一个参数改为字符串或整数数组")
				}
				indexType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if indexType != "整数" && indexType != "运行时整数" {
					return "", arrayIssue(current, "S-3034", "整数索引", indexType, "将数组索引改为整数表达式")
				}
				if indexType == "运行时整数" && !(strings.HasPrefix(arrayType, "运行时") && strings.HasSuffix(arrayType, "数组")) {
					return "", arrayIssue(current, "S-3034", "可变运行时数组的动态索引", arrayType, "将运行时索引应用于程序中的可变数组")
				}
				kind = strings.TrimSuffix(strings.TrimPrefix(arrayType, "运行时"), "数组")
				if strings.HasPrefix(arrayType, "运行时") || indexType == "运行时整数" {
					kind = "运行时" + kind
				}
			case "arraySet":
				if len(current.args) != 3 {
					return "", arrayIssue(current, "S-3035", "3 个参数", strconv.Itoa(len(current.args))+" 个参数", "提供可变整数数组、索引和值")
				}
				arrayType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if !(strings.HasPrefix(arrayType, "运行时") && strings.HasSuffix(arrayType, "数组")) {
					return "", arrayIssue(current, "S-3033", "可变数组", arrayType, "arraySet 只更新顶层变量中的运行时数组")
				}
				indexType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if indexType != "整数" && indexType != "运行时整数" {
					return "", arrayIssue(current, "S-3034", "整数索引", indexType, "将数组索引改为整数表达式")
				}
				valueType, problem := constantType(current.args[2], constants)
				if problem != nil {
					return "", problem
				}
				if strings.TrimSuffix(strings.TrimPrefix(arrayType, "运行时"), "数组") != strings.TrimPrefix(valueType, "运行时") {
					return "", arrayIssue(current, "S-3031", "同类数组元素", valueType, "写入与数组元素类型相同的值")
				}
				kind = arrayType
			case "arrayPush":
				if len(current.args) != 2 {
					return "", arrayIssue(current, "S-3035", "2 个参数", strconv.Itoa(len(current.args))+" 个参数", "提供可变整数数组和值")
				}
				arrayType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if !(strings.HasPrefix(arrayType, "运行时") && strings.HasSuffix(arrayType, "数组")) {
					return "", arrayIssue(current, "S-3033", "可变数组", arrayType, "arrayPush 只更新顶层变量中的运行时数组")
				}
				valueType, problem := constantType(current.args[1], constants)
				if problem != nil {
					return "", problem
				}
				if strings.TrimSuffix(strings.TrimPrefix(arrayType, "运行时"), "数组") != strings.TrimPrefix(valueType, "运行时") {
					return "", arrayIssue(current, "S-3031", "同类数组元素", valueType, "追加与数组元素类型相同的值")
				}
				kind = arrayType
			case "arrayLength":
				if len(current.args) != 1 {
					return "", arrayIssue(current, "S-3035", "1 个参数", strconv.Itoa(len(current.args))+" 个参数", "提供一个字符串或整数数组")
				}
				arrayType, problem := constantType(current.args[0], constants)
				if problem != nil {
					return "", problem
				}
				if arrayType != "字符串数组" && arrayType != "整数数组" && arrayType != "布尔值数组" && arrayType != "运行时整数数组" && arrayType != "运行时字符串数组" && arrayType != "运行时布尔值数组" {
					return "", arrayIssue(current, "S-3036", "数组", arrayType, "将参数改为字符串或整数数组")
				}
				kind = "整数"
				if strings.HasPrefix(arrayType, "运行时") {
					kind = "运行时整数"
				}
			default:
				return "", &issue{code: "S-3023", expected: "字符串或整数常量表达式", actual: current.text, suggestion: "使用当前支持的纯常量表达式", pos: current.pos}
			}
			types = append(types, kind)
		default:
			return "", &issue{code: "S-3023", expected: "字符串或整数常量表达式", actual: current.kind, suggestion: "使用普通字符串、整数或已声明常量", pos: current.pos}
		}
	}
	return types[0], nil
}
func runtimeTextConcatType(call value, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 2 {
		return "", &issue{code: "S-3044", expected: "2 个有界文本参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供 textConcat（左侧文本，右侧文本）", pos: call.pos}
	}
	for _, argument := range call.args {
		kind, problem := constantType(argument, constants)
		if problem != nil {
			return "", problem
		}
		if kind == "字符串" {
			text, textProblem := evaluateString(argument, 0, constants)
			if textProblem != nil {
				return "", textProblem
			}
			if len(text) > 63 {
				return "", &issue{code: "S-3044", expected: "最多 63 字节的静态文本操作数", actual: strconv.Itoa(len(text)) + " 字节", suggestion: "将静态文本缩短到有界文本缓冲区范围内", pos: argument.pos}
			}
			continue
		}
		if kind == "运行时字符串" && (argument.kind == "引用" || isRuntimeArgumentText(argument) || isTextResultValue(argument, constants) || argument.kind == "调用" && (argument.text == "readText" || argument.text == "arrayGet" || argument.text == "textConcat" || argument.text == "substring" && (argument.args[0].kind == "引用" || isRuntimeArgumentText(argument.args[0]) || isTextResultValue(argument.args[0], constants) || argument.args[0].kind == "调用" && (argument.args[0].text == "textConcat" || argument.args[0].text == "readText" || argument.args[0].text == "arrayGet")))) {
			continue
		}
		if kind != "运行时字符串" || argument.kind != "引用" {
			return "", &issue{code: "S-3044", expected: "已初始化的运行时文本变量、变量切片或静态字符串", actual: kind, suggestion: "使用运行时文本变量、以变量为源的 substring 或最多 63 字节的编译期字符串", pos: argument.pos}
		}
	}
	if runtimeTextConcatDepth(call) > 5 {
		return "", &issue{code: "S-3044", expected: "最多五层动态 textConcat 表达式", actual: "嵌套深度超过 5", suggestion: "先将内层拼接写入文本变量，再继续拼接", pos: call.pos}
	}
	return "运行时字符串", nil
}
func runtimeTextConcatDepth(input value) uint32 {
	maximum := uint32(0)
	pending := []struct {
		expression value; depth      uint32
	}{{expression: input}}
	for len(pending) > 0 {
		current := pending[len(pending)-1]; pending = pending[:len(pending)-1]
		if current.expression.kind == "调用" && current.expression.text == "textConcat" {
			current.depth++
			if current.depth > maximum {
				maximum = current.depth
			}
		}
		for _, argument := range current.expression.args {
			pending = append(pending, struct {
				expression value; depth      uint32
			}{argument, current.depth})
		}
		if current.expression.left != nil {
			pending = append(pending, struct {
				expression value; depth      uint32
			}{*current.expression.left, current.depth})
		}
		if current.expression.right != nil {
			pending = append(pending, struct {
				expression value; depth      uint32
			}{*current.expression.right, current.depth})
		}
		for _, item := range current.expression.fields {
			pending = append(pending, struct {
				expression value; depth      uint32
			}{item.value, current.depth})
		}
	}
	return maximum
}
func runtimeTextSubstringType(call value, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 3 {
		return "", &issue{code: "S-3011", expected: "3 个参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供文本、零起始起点和右端不含的终点", pos: call.pos}
	}
	textType, problem := constantType(call.args[0], constants)
	if problem != nil {
		return "", problem
	}
	if textType == "字符串" {
		return "字符串", nil
	}
	if textType != "运行时字符串" {
		return "", &issue{code: "S-3013", expected: "字符串或运行时文本变量", actual: textType, suggestion: "向 substring 提供字符串值", pos: call.args[0].pos}
	}
	if call.args[0].kind != "引用" && !isRuntimeArgumentText(call.args[0]) && !isTextResultValue(call.args[0], constants) && (call.args[0].kind != "调用" || call.args[0].text != "textConcat" && call.args[0].text != "arrayGet" && call.args[0].text != "readText") {
		return "", &issue{code: "S-3013", expected: "运行时文本变量或 textConcat 结果", actual: "运行时文本表达式", suggestion: "使用文本变量或单层有界 textConcat 结果作为切片源", pos: call.args[0].pos}
	}
	for _, operand := range call.args[0].args {
		if operand.kind == "调用" && operand.text == "substring" {
			return "", &issue{code: "S-3013", expected: "运行时变量与静态文本组成的单层 textConcat 源", actual: "嵌套切片拼接", suggestion: "先将切片或拼接结果写入变量，再执行下一步操作", pos: operand.pos}
		}
	}
	indices := [2]int64{}; staticIndex := [2]bool{}
	for index, argument := range call.args[1:] {
		kind, typeProblem := constantType(argument, constants)
		if typeProblem != nil {
			return "", typeProblem
		}
		if kind != "整数" && kind != "运行时整数" {
			return "", &issue{code: "S-3013", expected: "整数切片索引（0 至 63）", actual: kind, suggestion: "将切片索引改为整数值", pos: argument.pos}
		}
		if kind == "整数" {
			indices[index], problem = integerValue(argument, constants)
			if problem != nil {
				return "", problem
			}
			staticIndex[index] = true
			if indices[index] < 0 || indices[index] > 63 {
				return "", &issue{code: "S-3013", expected: "0 至 63 的切片索引", actual: strconv.FormatInt(indices[index], 10), suggestion: "使用有界文本缓冲区可表示的零起始码位索引", pos: argument.pos}
			}
		}
	}
	if staticIndex[0] && staticIndex[1] && indices[1] < indices[0] {
		return "", &issue{code: "S-3013", expected: "0 ≤ 起点 ≤ 终点 ≤ 63 个输入字节的最大码位数", actual: strconv.FormatInt(indices[0], 10) + ".." + strconv.FormatInt(indices[1], 10), suggestion: "使用有界文本缓冲区可表示的零起始码位索引", pos: call.pos}
	}
	return "运行时字符串", nil
}
func booleanCallType(call value, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 2 {
		return "", &issue{code: "S-3017", expected: "2 个比较参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "向 equals 提供两个同类型字符串、整数或布尔表达式", pos: call.pos}
	}
	left, problem := constantType(call.args[0], constants)
	if problem != nil {
		return "", problem
	}
	right, problem := constantType(call.args[1], constants)
	if problem != nil {
		return "", problem
	}
	leftIsText := left == "字符串" || left == "运行时字符串"; rightIsText := right == "字符串" || right == "运行时字符串"
	if leftIsText && rightIsText {
		if left == "运行时字符串" && !runtimeTextComparisonOperand(call.args[0]) || right == "运行时字符串" && !runtimeTextComparisonOperand(call.args[1]) {
			return "", &issue{code: "S-3015", expected: "受支持的运行时文本操作数或编译期字符串", actual: "运行时文本表达式", suggestion: "使用文本变量、readText、textConcat、substring 或文本数组元素", pos: call.pos}
		}
		if left == "运行时字符串" || right == "运行时字符串" {
			return "运行时布尔值", nil
		}
		return "布尔值", nil
	}
	if (left == "整数" || left == "运行时整数") && (right == "整数" || right == "运行时整数") || (left == "布尔值" || left == "运行时布尔值") && (right == "布尔值" || right == "运行时布尔值") {
		if left == "运行时整数" || right == "运行时整数" || left == "运行时布尔值" || right == "运行时布尔值" {
			return "运行时布尔值", nil
		}
		return "布尔值", nil
	}
	return "", &issue{code: "S-3015", expected: "两个同类型字符串、整数或布尔参数", actual: left + " 与 " + right, suggestion: "让 equals 两侧使用相同的字符串或整数类型", pos: call.pos}
}
func runtimeTextComparisonOperand(input value) bool {
	return input.kind == "引用" || isRuntimeArgumentText(input) || isTextReturnCall(input) || input.kind == "调用" &&
		(input.text == "readText" || input.text == "textConcat" || input.text == "substring" || input.text == "arrayGet" || input.text == "结果值" && len(input.args) == 1 && input.args[0].kind == "引用")
}
func isTextResultValue(input value, constants map[string]constantValue) bool {
	if input.kind != "调用" || input.text != "结果值" || len(input.args) != 1 || !isResultOperand(input.args[0]) {
		return false
	}
	return isTextResultPayload(runtimeResultPayload(input.args[0], constants))
}
func ifElseType(call value, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 3 {
		return "", &issue{code: "S-3016", expected: "3 个参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供条件、真分支值和假分支值", pos: call.pos}
	}
	condition, problem := constantType(call.args[0], constants)
	if problem != nil {
		return "", problem
	}
	if condition != "布尔值" && condition != "运行时布尔值" {
		return "", &issue{code: "S-3015", expected: "布尔比较或逻辑组合条件", actual: condition, suggestion: "使用整数比较、equals/notEquals 或且/故组合条件", pos: call.pos}
	}
	whenTrue, problem := constantType(call.args[1], constants)
	if problem != nil {
		return "", problem
	}
	whenFalse, problem := constantType(call.args[2], constants)
	if problem != nil {
		return "", problem
	}
	if whenTrue == "字符串" && whenFalse == "字符串" {
		if condition == "运行时布尔值" {
			return "", &issue{code: "S-3042", expected: "编译期确定的字符串分支条件", actual: "运行时整数比较", suggestion: "字符串 ifElse 暂只接受可静态求值的 equals 条件", pos: call.pos}
		}
		return "字符串", nil
	}
	if (whenTrue == "整数" || whenTrue == "运行时整数") && (whenFalse == "整数" || whenFalse == "运行时整数") {
		if condition == "运行时布尔值" || whenTrue == "运行时整数" || whenFalse == "运行时整数" {
			return "运行时整数", nil
		}
		return "整数", nil
	}
	if whenTrue == "运行时结果" && whenFalse == "运行时结果" {
		truePayload := runtimeResultPayload(call.args[1], constants); falsePayload := runtimeResultPayload(call.args[2], constants)
		if truePayload != "" && falsePayload != "" && truePayload != falsePayload {
			return "", &issue{code: "S-3023", expected: "相同的成功载荷类型", actual: truePayload + " 与 " + falsePayload, suggestion: "让 ifElse 两个分支返回相同类型的成功载荷", pos: call.pos}
		}
		return "运行时结果", nil
	}
	return "", &issue{code: "S-3023", expected: "两个同类型字符串或整数分支", actual: whenTrue + " 与 " + whenFalse, suggestion: "让 ifElse 的真、假分支返回相同类型", pos: call.pos}
}
func arrayIssue(call value, code, expected, actual, suggestion string) *issue {
	return &issue{code: code, expected: expected, actual: actual, suggestion: suggestion, pos: call.pos}
}
func evaluateArray(input value, constants map[string]constantValue) ([]constantValue, *issue) {
	if input.kind == "引用" {
		constant, exists := constants[input.text]
		if !exists {
			return nil, unresolvedConstant(input)
		}
		if constant.kind != "字符串数组" && constant.kind != "整数数组" && constant.kind != "运行时整数数组" {
			return nil, arrayIssue(input, "S-3033", "数组", constant.kind, "引用字符串或整数数组常量")
		}
		return constant.array, nil
	}
	if input.kind != "调用" || input.text != "array" {
		return nil, arrayIssue(input, "S-3033", "数组", input.kind, "使用 array（元素，…）构造数组")
	}
	if len(input.args) == 0 {
		return nil, arrayIssue(input, "S-3030", "至少一个数组元素", "空数组", "补入一个或多个字符串或整数元素")
	}
	elementType, problem := constantType(input.args[0], constants)
	if problem != nil {
		return nil, problem
	}
	baseType := strings.TrimPrefix(elementType, "运行时")
	if baseType != "字符串" && baseType != "整数" || baseType == "字符串" && elementType != "字符串" {
		return nil, arrayIssue(input, "S-3031", "字符串或整数元素", elementType, "数组不接受嵌套数组")
	}
	items := make([]constantValue, 0, len(input.args))
	for _, argument := range input.args {
		kind, issue := constantType(argument, constants)
		if issue != nil {
			return nil, issue
		}
		if strings.TrimPrefix(kind, "运行时") != baseType {
			return nil, arrayIssue(input, "S-3031", baseType+" 数组元素", kind, "数组中所有元素必须为同一类型")
		}
		item := constantValue{kind: kind}
		if kind == "字符串" {
			item.text, issue = evaluateString(argument, 0, constants)
		} else if kind == "整数" {
			item.integer, issue = integerValue(argument, constants)
		}
		if issue != nil {
			return nil, issue
		}
		items = append(items, item)
	}
	return items, nil
}
func evaluateArrayAccess(call value, constants map[string]constantValue) (constantValue, *issue) {
	if len(call.args) != 2 {
		return constantValue{}, arrayIssue(call, "S-3035", "2 个参数", strconv.Itoa(len(call.args))+" 个参数", "提供数组和零起始整数索引")
	}
	arrayType, problem := constantType(call.args[0], constants)
	if problem != nil {
		return constantValue{}, problem
	}
	if arrayType != "字符串数组" && arrayType != "整数数组" && arrayType != "布尔值数组" && arrayType != "运行时整数数组" && arrayType != "运行时布尔值数组" {
		return constantValue{}, arrayIssue(call, "S-3033", "数组", arrayType, "将第一个参数改为字符串或整数数组")
	}
	indexType, problem := constantType(call.args[1], constants)
	if problem != nil {
		return constantValue{}, problem
	}
	if indexType != "整数" {
		return constantValue{}, arrayIssue(call, "S-3034", "整数索引", indexType, "将数组索引改为整数表达式")
	}
	items, problem := evaluateArray(call.args[0], constants)
	if problem != nil {
		return constantValue{}, problem
	}
	index, problem := integerValue(call.args[1], constants)
	if problem != nil {
		return constantValue{}, problem
	}
	if index < 0 || index >= int64(len(items)) {
		return constantValue{}, arrayIssue(call, "S-3032", "0 ≤ 索引 < 数组长度", strconv.FormatInt(index, 10)+" / "+strconv.Itoa(len(items)), "将索引调整到数组的有效范围内")
	}
	return items[int(index)], nil
}
func evaluateArrayLength(call value, constants map[string]constantValue) (int64, *issue) {
	if len(call.args) != 1 {
		return 0, arrayIssue(call, "S-3035", "1 个参数", strconv.Itoa(len(call.args))+" 个参数", "提供一个字符串或整数数组")
	}
	arrayType, problem := constantType(call.args[0], constants)
	if problem != nil {
		return 0, problem
	}
	if arrayType != "字符串数组" && arrayType != "整数数组" && arrayType != "布尔值数组" && arrayType != "运行时整数数组" && arrayType != "运行时布尔值数组" {
		return 0, arrayIssue(call, "S-3036", "数组", arrayType, "将参数改为字符串或整数数组")
	}
	if call.args[0].kind == "调用" && call.args[0].text == "array" {
		return int64(len(call.args[0].args)), nil
	}
	items, problem := evaluateArray(call.args[0], constants)
	if problem != nil {
		return 0, problem
	}
	return int64(len(items)), nil
}
func unresolvedConstant(input value) *issue {
	return &issue{code: "S-3022", expected: "此前声明的常量或变量", actual: input.text, suggestion: "先声明该名称；常量和变量引用均不能向前查找", pos: input.pos}
}
func evaluateString(input value, depth int, constants map[string]constantValue) (string, *issue) {
	if depth > 64 {
		return "", &issue{code: "S-3014", expected: "不超过 64 层的字符串调用", actual: "调用嵌套过深", suggestion: "拆分嵌套的字符串表达式", pos: input.pos}
	}
	stack := []value{input}; var output strings.Builder
	for len(stack) > 0 {
		index := len(stack) - 1; current := stack[index]
		stack = stack[:index]
		if current.kind == "字符串" {
			output.WriteString(current.text)
			continue
		}
		if current.kind == "引用" {
			constant, exists := constants[current.text]
			if !exists {
				return "", unresolvedConstant(current)
			}
			if constant.kind != "字符串" {
				return "", &issue{code: "S-3023", expected: "字符串常量", actual: "整数常量 " + current.text, suggestion: "在文本表达式中引用字符串常量", pos: current.pos}
			}
			output.WriteString(constant.text)
			continue
		}
		if current.kind == "调用" {
			text, problem := evaluateStringCall(current, depth+1, constants)
			if problem != nil {
				return "", problem
			}
			output.WriteString(text)
			continue
		}
		if current.kind != "表达式" || current.operator != "+" || current.left == nil || current.right == nil {
			return "", &issue{code: "S-3005", expected: "字符串字面量、字符串拼接、substring 或 ifElse 调用", actual: current.kind, suggestion: "使用『普通字符串』或支持的字符串表达式", pos: current.pos}
		}
		stack = append(stack, *current.right, *current.left)
	}
	return output.String(), nil
}
func evaluateStringCall(call value, depth int, constants map[string]constantValue) (string, *issue) {
	switch call.text {
	case "substring":
		return evaluateSubstring(call, depth, constants)
	case "ifElse":
		return evaluateIfElse(call, depth, constants)
	case "arrayGet":
		item, problem := evaluateArrayAccess(call, constants)
		if problem != nil {
			return "", problem
		}
		if item.kind != "字符串" {
			return "", arrayIssue(call, "S-3012", "字符串数组元素", item.kind, "在文本表达式中使用字符串数组")
		}
		return item.text, nil
	default:
		return "", &issue{code: "S-3010", expected: "substring 或 ifElse 字符串函数", actual: call.text, suggestion: "使用当前已实现的字符串函数", pos: call.pos}
	}
}
func evaluateSubstring(call value, depth int, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 3 {
		return "", &issue{code: "S-3011", expected: "3 个参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供文本、零起始起点和右端不含的终点", pos: call.pos}
	}
	source, problem := evaluateString(call.args[0], depth, constants)
	if problem != nil {
		return "", problem
	}
	start, problem := integerValue(call.args[1], constants)
	if problem != nil {
		return "", problem
	}
	end, problem := integerValue(call.args[2], constants)
	if problem != nil {
		return "", problem
	}
	runes := []rune(source); length := int64(len(runes))
	if start < 0 || end < start || end > length {
		return "", &issue{code: "S-3013", expected: "0 ≤ 起点 ≤ 终点 ≤ 字符串码位数", actual: strconv.FormatInt(start, 10) + ".." + strconv.FormatInt(end, 10) + " / " + strconv.FormatInt(length, 10), suggestion: "将索引调整到字符串的零起始码位范围内", pos: call.pos}
	}
	return string(runes[int(start):int(end)]), nil
}
func evaluateIfElse(call value, depth int, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 3 {
		return "", &issue{code: "S-3016", expected: "3 个参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供条件、条件为真时的文本和条件为假时的文本", pos: call.pos}
	}
	condition, problem := evaluateBoolean(call.args[0], depth, constants)
	if problem != nil {
		return "", problem
	}
	whenTrue, problem := evaluateString(call.args[1], depth, constants)
	if problem != nil {
		return "", problem
	}
	whenFalse, problem := evaluateString(call.args[2], depth, constants)
	if problem != nil {
		return "", problem
	}
	if condition {
		return whenTrue, nil
	}
	return whenFalse, nil
}
func evaluateBoolean(condition value, depth int, constants map[string]constantValue) (bool, *issue) {
	if depth > 64 {
		return false, &issue{code: "S-3014", expected: "不超过 64 层的字符串调用", actual: "调用嵌套过深", suggestion: "拆分嵌套的条件表达式", pos: condition.pos}
	}
	if condition.kind == "逻辑" {
		if condition.left == nil || condition.right == nil {
			return false, &issue{code: "S-3015", expected: "完整的且/故布尔表达式", actual: condition.operator, suggestion: "在且、故两侧提供布尔条件", pos: condition.pos}
		}
		left, problem := evaluateBoolean(*condition.left, depth+1, constants)
		if problem != nil {
			return false, problem
		}
		if condition.operator == "且" && !left || condition.operator == "故" && left {
			return left, nil
		}
		if condition.operator != "且" && condition.operator != "故" {
			return false, &issue{code: "S-3015", expected: "且 或 故", actual: condition.operator, suggestion: "使用且表达逻辑与，使用故表达逻辑或", pos: condition.pos}
		}
		return evaluateBoolean(*condition.right, depth+1, constants)
	}
	if condition.kind == "比较" {
		kind, problem := constantType(condition, constants)
		if problem != nil {
			return false, problem
		}
		if kind != "布尔值" {
			return false, &issue{code: "S-3042", expected: "可静态求值的整数比较", actual: "运行时整数比较", suggestion: "在整数 ifElse 中使用运行时比较", pos: condition.pos}
		}
		leftType, _ := constantType(*condition.left, constants)
		if leftType == "字符串" {
			left, problem := evaluateString(*condition.left, depth+1, constants)
			if problem != nil {
				return false, problem
			}
			right, problem := evaluateString(*condition.right, depth+1, constants)
			if problem != nil {
				return false, problem
			}
			return compareStringValues(condition.operator, strings.Compare(left, right)), nil
		}
		left, problem := integerValue(*condition.left, constants)
		if problem != nil {
			return false, problem
		}
		right, problem := integerValue(*condition.right, constants)
		if problem != nil {
			return false, problem
		}
		return compareIntegerValues(condition.operator, left, right), nil
	}
	if condition.kind != "调用" || condition.text != "equals" && condition.text != "notEquals" {
		return false, &issue{code: "S-3015", expected: "布尔比较或逻辑组合条件", actual: condition.kind, suggestion: "使用整数比较、equals/notEquals 或且/故组合条件", pos: condition.pos}
	}
	kind, problem := booleanCallType(condition, constants)
	if problem != nil {
		return false, problem
	}
	if kind == "布尔值" {
		leftType, _ := constantType(condition.args[0], constants)
		if leftType == "字符串" {
			left, problem := evaluateString(condition.args[0], depth+1, constants)
			if problem != nil {
				return false, problem
			}
			right, problem := evaluateString(condition.args[1], depth+1, constants)
			return (left == right) != (condition.text == "notEquals"), problem
		}
		if leftType == "布尔值" {
			left, problem := evaluateBoolean(condition.args[0], depth+1, constants)
			if problem != nil {
				return false, problem
			}
			right, problem := evaluateBoolean(condition.args[1], depth+1, constants)
			return (left == right) != (condition.text == "notEquals"), problem
		}
		left, problem := integerValue(condition.args[0], constants)
		if problem != nil {
			return false, problem
		}
		right, problem := integerValue(condition.args[1], constants)
		return (left == right) != (condition.text == "notEquals"), problem
	}
	return false, &issue{code: "S-3042", expected: "可静态求值的 equals 条件", actual: "运行时整数比较", suggestion: "仅在动态整数 ifElse 中使用运行时比较", pos: condition.pos}
}
func compareStringValues(operator string, order int) bool {
	switch operator {
	case "<":
		return order < 0
	case "<=":
		return order <= 0
	case ">":
		return order > 0
	case ">=":
		return order >= 0
	default:
		return false
	}
}
func integerValue(input value, constants map[string]constantValue) (int64, *issue) {
	type integerFrame struct {
		current value; visited bool
	}
	frames := []integerFrame{{current: input}}; values := []int64{}
	for len(frames) > 0 {
		index := len(frames) - 1; frame := frames[index]
		frames = frames[:index]; current := frame.current
		if current.kind == "整数" {
			number, err := strconv.ParseInt(current.text, 10, 64)
			if err != nil {
				return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: current.text, suggestion: "将整数缩小到 -9223372036854775808 至 9223372036854775807", pos: current.pos}
			}
			values = append(values, number)
			continue
		}
		if current.kind == "引用" {
			constant, exists := constants[current.text]
			if !exists {
				return 0, unresolvedConstant(current)
			}
			if constant.kind != "整数" {
				return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: "字符串常量 " + current.text, suggestion: "在整数表达式中引用整数常量", pos: current.pos}
			}
			values = append(values, constant.integer)
			continue
		}
		if current.kind == "比较" || current.kind == "逻辑" {
			return 0, &issue{code: "S-3023", expected: "整数表达式", actual: "布尔条件 " + current.operator, suggestion: "将比较或逻辑表达式放在 ifElse、若或凡的条件位置", pos: current.pos}
		}
		if current.kind == "表达式" {
			if current.left == nil || (current.operator != "negate" && current.right == nil) {
				return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: current.operator, suggestion: "索引表达式需要完整的整数运算操作数", pos: current.pos}
			}
			if !frame.visited {
				frames = append(frames, integerFrame{current: current, visited: true})
				if current.right != nil {
					frames = append(frames, integerFrame{current: *current.right})
				}
				frames = append(frames, integerFrame{current: *current.left})
				continue
			}
			if current.operator == "negate" {
				operand := values[len(values)-1]; values = values[:len(values)-1]
				if operand == -1<<63 {
					return 0, &issue{code: "S-3018", expected: "范围内的有符号 64 位整数结果", actual: "-(-9223372036854775808)", suggestion: "避免对最小有符号整数取负", pos: current.pos}
				}
				values = append(values, -operand)
				continue
			}
			right := values[len(values)-1]; left := values[len(values)-2]
			values = values[:len(values)-2]
			if current.operator == "<<" || current.operator == ">>" {
				if right < 0 || right >= 64 {
					return 0, &issue{code: "S-3041", expected: "0 至 63 的移位量", actual: strconv.FormatInt(right, 10), suggestion: "将移位量限制在 64 位整数的有效范围内", pos: current.pos}
				}
			}
			result, valid := checkedIntegerArithmetic(current.operator, left, right)
			if (current.operator == "/" || current.operator == "%") && right == 0 {
				return 0, &issue{code: "S-3024", expected: "非零除数", actual: "0", suggestion: "将除数改为非零整数常量", pos: current.pos}
			}
			if !valid {
				actual := strconv.FormatInt(left, 10) + " " + current.operator + " " + strconv.FormatInt(right, 10)
				return 0, &issue{code: "S-3018", expected: "范围内的有符号 64 位整数结果", actual: actual, suggestion: "缩小整数运算结果以避免溢出", pos: current.pos}
			}
			values = append(values, result)
			continue
		}
		if current.kind == "调用" {
			if current.text == "readInteger" {
				return 0, &issue{code: "S-3042", expected: "可静态求值的整数表达式", actual: "readInteger（）运行时值", suggestion: "运行时整数不能参与编译期求值或动态算术", pos: current.pos}
			}
			if current.text == "textByteLength" || current.text == "textByteAt" {
				result, problem := staticTextByteValue(current, constants)
				if problem != nil {
					return 0, problem
				}
				values = append(values, result)
				continue
			}
			if current.text == "textLength" {
				if len(current.args) != 1 {
					return 0, &issue{code: "S-3044", expected: "1 个文本长度参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供一个静态字符串表达式", pos: current.pos}
				}
				text, problem := evaluateString(current.args[0], 0, constants)
				if problem != nil {
					return 0, problem
				}
				values = append(values, int64(len([]rune(text))))
				continue
			}
			if current.text == "ifElse" {
				if len(current.args) != 3 {
					return 0, &issue{code: "S-3016", expected: "3 个参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供条件、真分支值和假分支值", pos: current.pos}
				}
				condition, problem := evaluateBoolean(current.args[0], 0, constants)
				if problem != nil {
					return 0, problem
				}
				selected := 1
				if !condition {
					selected = 2
				}
				frames = append(frames, integerFrame{current: current.args[selected]})
				continue
			}
			if current.text == "arrayLength" {
				length, problem := evaluateArrayLength(current, constants)
				if problem != nil {
					return 0, problem
				}
				values = append(values, length)
				continue
			}
			if current.text == "arrayGet" {
				item, problem := evaluateArrayAccess(current, constants)
				if problem != nil {
					return 0, problem
				}
				if item.kind != "整数" {
					return 0, arrayIssue(current, "S-3023", "整数数组元素", item.kind, "在整数表达式中使用整数数组")
				}
				values = append(values, item.integer)
				continue
			}
			if current.text != "bitAnd" && current.text != "bitOr" && current.text != "bitXor" {
				return 0, &issue{code: "S-3020", expected: "bitAnd、bitOr 或 bitXor 整数函数", actual: current.text, suggestion: "使用当前已实现的位运算函数", pos: current.pos}
			}
			if len(current.args) != 2 {
				return 0, &issue{code: "S-3021", expected: "2 个位运算参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供两个有符号 64 位整数常量表达式", pos: current.pos}
			}
			if !frame.visited {
				frames = append(frames, integerFrame{current: current, visited: true}, integerFrame{current: current.args[1]}, integerFrame{current: current.args[0]})
				continue
			}
			right := values[len(values)-1]; left := values[len(values)-2]
			values = values[:len(values)-2]; result := int64(0)
			switch current.text {
			case "bitAnd":
				result = left & right
			case "bitOr":
				result = left | right
			case "bitXor":
				result = left ^ right
			}
			values = append(values, result)
			continue
		}
		return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: current.kind, suggestion: "索引表达式使用有符号 64 位整数和当前支持的运算", pos: current.pos}
	}
	return values[0], nil
}
func checkedIntegerArithmetic(operator string, left, right int64) (int64, bool) {
	maxInt64 := int64(1<<63 - 1); minInt64 := int64(-1 << 63)
	switch operator {
	case "+":
		if right > 0 && left > maxInt64-right || right < 0 && left < minInt64-right {
			return 0, false
		}
		return left + right, true
	case "-":
		if right < 0 && left > maxInt64+right || right > 0 && left < minInt64+right {
			return 0, false
		}
		return left - right, true
	case "*":
		if left == 0 || right == 0 {
			return 0, true
		}
		if left == minInt64 && right == -1 || right == minInt64 && left == -1 {
			return 0, false
		}
		product := left * right
		if product/right != left {
			return 0, false
		}
		return product, true
	case "/":
		if right == 0 || left == minInt64 && right == -1 {
			return 0, false
		}
		return left / right, true
	case "%":
		if right == 0 {
			return 0, false
		}
		if left == minInt64 && right == -1 {
			return 0, true
		}
		return left % right, true
	case "<<":
		if right < 0 || right >= 64 {
			return 0, false
		}
		shifted := uint64(left) << uint(right)
		if int64(shifted)>>uint(right) != left {
			return 0, false
		}
		return int64(shifted), true
	case ">>":
		if right < 0 || right >= 64 {
			return 0, false
		}
		return left >> uint(right), true
	default:
		return 0, false
	}
}
func compareIntegerValues(operator string, left, right int64) bool {
	switch operator {
	case "==":
		return left == right
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case ">=":
		return left >= right
	default:
		return false
	}
}
func allowsRuntimeTextReturnCall(item statement) bool {
	return (item.kind == "变量" || item.kind == "赋值") &&
		(isTextReturnCall(item.value) || runtimeTextReturnCallAsBorrowedTextArgument(item.value))
}
func isRuntimeTextReturnBorrowConsumer(input value) bool {
	if input.kind != "调用" {
		return false
	}
	switch input.text {
	case "textLength", "equals", "notEquals":
		return true
	}
	return false
}
func runtimeTextReturnCallAsBorrowedTextArgument(input value) bool {
	if input.kind != "调用" {
		return false
	}
	if input.text == "ifElse" && len(input.args) == 3 {
		condition := input.args[0]
		if containsRuntimeTextReturnCall(condition) {
			if !(condition.kind == "调用" && isRuntimeTextReturnBorrowConsumer(condition) && runtimeTextReturnCallAsBorrowedTextArgument(condition)) {
				return false
			}
		}
		return !containsRuntimeTextReturnCall(input.args[1]) && !containsRuntimeTextReturnCall(input.args[2])
	}
	for _, argument := range input.args {
		if isTextReturnCall(argument) {
			continue
		}
		if containsRuntimeTextReturnCall(argument) {
			return false
		}
	}
	return valuesContainRuntimeTextReturnCall(input.args)
}
