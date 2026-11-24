// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main; import ("fmt"; "os"; "os/exec"; "path/filepath"; "runtime"; "strconv"; "strings")
const (runtimeTextConcatBaseScratch   uint32 = 512; runtimeTextConcatScratchStride uint32 = 512); const syscallInstruction uint32 = 0xd4001001
type conditionalPatch struct {
	position  int; condition uint32
}
type conditionFalsePatch struct {
	position      int; condition     uint32
	unconditional bool
}
func isRuntimeIntegerInput(expression value) bool { return expression.kind == "调用" && expression.text == "readInteger" && len(expression.args) == 0 }
func isRuntimeTextInput(expression value) bool { return expression.kind == "调用" && expression.text == "readText" && len(expression.args) == 0 }
func isRuntimeArgumentCount(expression value) bool { return expression.kind == "调用" && expression.text == "argumentCount" && len(expression.args) == 0 }
func isRuntimeArgumentText(expression value) bool { return expression.kind == "调用" && expression.text == "argumentText" && len(expression.args) == 1 }
func isRuntimeEnvironmentRead(expression value) bool { return expression.kind == "调用" && expression.text == "readEnvironment" && len(expression.args) == 1 }
func isRuntimeEnvironmentNames(expression value) bool { return expression.kind == "调用" && expression.text == "environmentNames" && len(expression.args) == 0 }
func isRuntimeFileRead(expression value) bool { return expression.kind == "调用" && (expression.text == "readFile" && len(expression.args) == 1 || expression.text == "readModule" && isModuleRead(expression)) }
func isRuntimeFileWrite(expression value) bool { return expression.kind == "调用" && (expression.text == "writeFile" || expression.text == "appendFile" || expression.text == "writeByteArray") && len(expression.args) == 2 }
func isRuntimeFileSize(expression value) bool { return expression.kind == "调用" && expression.text == "fileSize" && len(expression.args) == 1 }
func isRuntimePathExists(expression value) bool { return expression.kind == "调用" && expression.text == "exists" && len(expression.args) == 1 }
func isRuntimeDirectoryCheck(expression value) bool { return expression.kind == "调用" && expression.text == "isDirectory" && len(expression.args) == 1 }
func isRuntimeFileRemove(expression value) bool { return expression.kind == "调用" && expression.text == "removeFile" && len(expression.args) == 1 }
func isRuntimeFileRename(expression value) bool { return expression.kind == "调用" && expression.text == "renameFile" && len(expression.args) == 2 }
func isRuntimeFileReplace(expression value) bool { return expression.kind == "调用" && expression.text == "replaceFile" && len(expression.args) == 2 }
func isRuntimeDirectoryMutation(expression value) bool { return expression.kind == "调用" && (expression.text == "createDirectory" || expression.text == "removeDirectory") && len(expression.args) == 1 }
func isRuntimeFileMode(expression value) bool { return expression.kind == "调用" && (expression.text == "setFileMode" || expression.text == "changeFileMode") && len(expression.args) == 2 }
func containsRuntimeFileRead(expression value, moduleOnly bool) bool {
	if isModuleRead(expression) || !moduleOnly && isRuntimeFileRead(expression) {
		return true
	}
	if expression.left != nil && containsRuntimeFileRead(*expression.left, moduleOnly) || expression.right != nil && containsRuntimeFileRead(*expression.right, moduleOnly) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeFileRead(argument, moduleOnly) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeFileRead(statements []statement, moduleOnly bool) bool {
	for _, item := range statements {
		if containsRuntimeFileRead(item.value, moduleOnly) || containsRuntimeFileRead(item.condition, moduleOnly) || statementsUseRuntimeFileRead(item.body, moduleOnly) || statementsUseRuntimeFileRead(item.otherwise, moduleOnly) {
			return true
		}
	}
	return false
}
func containsRuntimeFileNamed(expression value, name string) bool {
	matches := isRuntimeFileWrite(expression) && (name == "write" || name == "!" && expression.text != "appendFile" || name != "write" && name != "!" && expression.text == name) ||
		(isRuntimeFileRemove(expression) || isRuntimeFileRename(expression) || isRuntimeFileReplace(expression) || isRuntimeDirectoryMutation(expression) || isRuntimeFileMode(expression) || isRuntimeFileSize(expression) || isRuntimePathExists(expression) || isRuntimeDirectoryCheck(expression)) && expression.text == name
	if matches {
		return true
	}
	if expression.left != nil && containsRuntimeFileNamed(*expression.left, name) || expression.right != nil && containsRuntimeFileNamed(*expression.right, name) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeFileNamed(argument, name) {
			return true
		}
	}
	return false
}
func containsRuntimeFileAppend(v value) bool { return containsRuntimeFileNamed(v, "appendFile") }
func containsRuntimeFileCreate(v value) bool { return containsRuntimeFileNamed(v, "!") }
func containsRuntimeFileRemove(v value) bool { return containsRuntimeFileNamed(v, "removeFile") }
func containsRuntimeFileRename(v value) bool { return containsRuntimeFileNamed(v, "renameFile") }
func statementsUseRuntimeFileNamed(statements []statement, name string) bool {
	for _, item := range statements {
		if containsRuntimeFileNamed(item.value, name) || containsRuntimeFileNamed(item.condition, name) || statementsUseRuntimeFileNamed(item.body, name) || statementsUseRuntimeFileNamed(item.otherwise, name) {
			return true
		}
	}
	return false
}
func statementsUseAppend(s []statement) bool            { return statementsUseRuntimeFileNamed(s, "appendFile") }
func statementsUseRuntimeFileCreate(s []statement) bool { return statementsUseRuntimeFileNamed(s, "!") }
func containsRuntimeArguments(expression value) bool {
	if isRuntimeArgumentCount(expression) || isRuntimeArgumentText(expression) {
		return true
	}
	if expression.left != nil && containsRuntimeArguments(*expression.left) || expression.right != nil && containsRuntimeArguments(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeArguments(argument) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeArguments(statements []statement) bool {
	for _, item := range statements {
		if containsRuntimeArguments(item.value) || containsRuntimeArguments(item.condition) || statementsUseRuntimeArguments(item.body) || statementsUseRuntimeArguments(item.otherwise) {
			return true
		}
	}
	return false
}
func containsRuntimeEnvironmentRead(expression value) bool {
	if isRuntimeEnvironmentRead(expression) || expression.left != nil && containsRuntimeEnvironmentRead(*expression.left) || expression.right != nil && containsRuntimeEnvironmentRead(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeEnvironmentRead(argument) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeEnvironmentRead(statements []statement) bool {
	for _, item := range statements {
		if containsRuntimeEnvironmentRead(item.value) || containsRuntimeEnvironmentRead(item.condition) || statementsUseRuntimeEnvironmentRead(item.body) || statementsUseRuntimeEnvironmentRead(item.otherwise) {
			return true
		}
	}
	return false
}
func containsRuntimeEnvironmentNames(expression value) bool {
	if isRuntimeEnvironmentNames(expression) || expression.left != nil && containsRuntimeEnvironmentNames(*expression.left) || expression.right != nil && containsRuntimeEnvironmentNames(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeEnvironmentNames(argument) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeEnvironmentNames(statements []statement) bool {
	for _, item := range statements {
		if containsRuntimeEnvironmentNames(item.value) || containsRuntimeEnvironmentNames(item.condition) || statementsUseRuntimeEnvironmentNames(item.body) || statementsUseRuntimeEnvironmentNames(item.otherwise) {
			return true
		}
	}
	return false
}
func runtimeFunctionsUseEnvironmentNames(runtimeFunctions map[string]functionDecl) bool {
	for _, function := range runtimeFunctions {
		if containsRuntimeEnvironmentNames(function.body) || statementsUseRuntimeEnvironmentNames(function.statements) {
			return true
		}
	}
	return false
}
func runtimeFunctionsUseEnvironmentRead(runtimeFunctions map[string]functionDecl) bool {
	for _, function := range runtimeFunctions {
		if containsRuntimeEnvironmentRead(function.body) || statementsUseRuntimeEnvironmentRead(function.statements) {
			return true
		}
	}
	return false
}
func isRuntimeTextValue(expression value) bool { return isRuntimeTextInput(expression) || isRuntimeEnvironmentRead(expression) || isRuntimeEnvironmentNames(expression) || expression.kind == "调用" && (expression.text == "textConcat" || expression.text == "substring" || expression.text == "arrayGet" || isTextReturnCall(expression)) }
func isStoredTextReturnCall(expression value) bool { return isTextReturnCall(expression) }
func containsRuntimeIntegerInput(expression value) bool {
	if isRuntimeIntegerInput(expression) {
		return true
	}
	if expression.left != nil && containsRuntimeIntegerInput(*expression.left) || expression.right != nil && containsRuntimeIntegerInput(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeIntegerInput(argument) {
			return true
		}
	}
	return false
}
func containsRuntimeTextInput(expression value) bool {
	if isRuntimeTextInput(expression) {
		return true
	}
	if expression.left != nil && containsRuntimeTextInput(*expression.left) || expression.right != nil && containsRuntimeTextInput(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeTextInput(argument) {
			return true
		}
	}
	return false
}
func containsTextResultValue(expression value) bool {
	if expression.kind == "调用" && expression.text == "结果值" && len(expression.args) == 1 && expression.args[0].kind == "引用" {
		return true
	}
	if expression.left != nil && containsTextResultValue(*expression.left) || expression.right != nil && containsTextResultValue(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsTextResultValue(argument) {
			return true
		}
	}
	return false
}
func isRuntimeResultScalar(expression value) bool {
	if expression.kind != "调用" {
		return false
	}
	if expression.runtimeFunction && expression.runtimeFunctionResult {
		return true
	}
	if expression.text == "结果值" || expression.text == "错误码" || expression.text == "是否失败" {
		return true
	}
	if len(expression.args) != 1 {
		return false
	}
	if expression.text == "成功" {
		return containsResultStateCall(expression) == false && !containsRuntimeTextInput(expression)
	}
	return expression.text == "失败"
}
func containsResultStateCall(expression value) bool {
	if expression.kind == "调用" && (expression.text == "是否失败" || expression.text == "错误码" || expression.text == "结果值") && len(expression.args) == 1 {
		return true
	}
	if expression.left != nil && containsResultStateCall(*expression.left) || expression.right != nil && containsResultStateCall(*expression.right) {
		return true
	}
	for _, argument := range expression.args {
		if containsResultStateCall(argument) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeArray(statements []statement) bool {
	for _, item := range statements {
		if item.kind == "变量" && item.value.kind == "调用" && item.value.text == "array" ||
			item.kind == "赋值" && item.value.kind == "调用" && item.value.text == "arrayPush" ||
			statementsUseRuntimeArray(item.body) || statementsUseRuntimeArray(item.otherwise) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeMap(statements []statement) bool {
	for _, item := range statements {
		if item.kind == "变量" && isRuntimeMapValue(item.value) || item.kind == "赋值" && item.value.kind == "调用" && item.value.text == "映射置" || item.kind == "删映射键" || statementsUseRuntimeMap(item.body) || statementsUseRuntimeMap(item.otherwise) {
			return true
		}
	}
	return false
}
func isTextResultExtraction(expression value) bool { return expression.kind == "调用" && expression.text == "结果值" && len(expression.args) == 1 && expression.args[0].kind == "引用" }
func isDirectTextResultExtraction(expression value) bool {
	return expression.kind == "调用" && expression.text == "结果值" && len(expression.args) == 1 &&
		expression.args[0].kind != "引用" && isResultOperand(expression.args[0])
}
func containsDirectRuntimeTextResultExtraction(expression value, constants map[string]constantValue) bool {
	found := false
	walkValue(expression, func(current value) bool {
		if isDirectTextResultExtraction(current) && runtimeResultPayload(current.args[0], constants) == "运行时文本" {
			found = true
		}
		return !found
	})
	return found
}
func statementsUseRuntimeTextStorage(statements []statement) bool {
	for _, item := range statements {
		if (item.kind == "变量" || item.kind == "赋值") && (isRuntimeTextValue(item.value) || isRuntimeFileRead(item.value) || isRuntimeArgumentText(item.value) || isTextResultExtraction(item.value) || runtimeTextResultExpression(item.value) || isStoredTextReturnCall(item.value) || isRuntimeResultValue(item.value) && !isRuntimeResultScalar(item.value)) ||
			statementsUseRuntimeTextStorage(item.body) || statementsUseRuntimeTextStorage(item.otherwise) {
			return true
		}
	}
	return false
}
func statementsUseOnlyBorrowedEnvironmentText(statements []statement) bool {
	used := false
	for _, item := range statements {
		if (item.kind == "变量" || item.kind == "赋值") && isRuntimeEnvironmentRead(item.value) {
			used = true
			continue
		}
		if statementsUseRuntimeTextStorage([]statement{item}) {
			return false
		}
	}
	return used
}
func runtimeFunctionsUseFileRead(runtimeFunctions map[string]functionDecl, moduleOnly bool) bool {
	for _, function := range runtimeFunctions {
		if containsRuntimeFileRead(function.body, moduleOnly) || statementsUseRuntimeFileRead(function.statements, moduleOnly) {
			return true
		}
	}
	return false
}
func runtimeFunctionsUseFileNamed(runtimeFunctions map[string]functionDecl, name string) bool {
	for _, function := range runtimeFunctions {
		if containsRuntimeFileNamed(function.body, name) || statementsUseRuntimeFileNamed(function.statements, name) {
			return true
		}
	}
	return false
}
func runtimeAppend(f map[string]functionDecl) bool { return runtimeFunctionsUseFileNamed(f, "appendFile") }
func runtimeFileCreate(f map[string]functionDecl) bool { return runtimeFunctionsUseFileNamed(f, "!") }
func runtimeFileRemove(f map[string]functionDecl) bool { return runtimeFunctionsUseFileNamed(f, "removeFile") }
func runtimeFileRename(f map[string]functionDecl) bool { return runtimeFunctionsUseFileNamed(f, "renameFile") }
func runtimeFileReplace(f map[string]functionDecl) bool { return runtimeFunctionsUseFileNamed(f, "replaceFile") }
func runtimeFunctionsUseArguments(runtimeFunctions map[string]functionDecl) bool {
	for _, function := range runtimeFunctions {
		if containsRuntimeArguments(function.body) || statementsUseRuntimeArguments(function.statements) {
			return true
		}
	}
	return false
}
func runtimeFunctionsUseMappedMemory(runtimeFunctions map[string]functionDecl) bool {
	if runtimeFunctionsUseTextLocals(runtimeFunctions) {
		return true
	}
	for _, function := range runtimeFunctions {
		if statementsUseRuntimeArray(function.statements) || statementsUseRuntimeMap(function.statements) || statementsUseRuntimeTextStorage(function.statements) {
			return true
		}
	}
	return false
}
func runtimeFunctionsUseSyscall(runtimeFunctions map[string]functionDecl, name string) bool {
	for _, function := range runtimeFunctions {
		values := []value{function.body}
		for _, item := range function.statements {
			values = append(values, item.value, item.condition)
		}
		for _, input := range values {
			stack := []value{input}
			for len(stack) > 0 {
				current := stack[len(stack)-1]; stack = stack[:len(stack)-1]
				if current.kind == "调用" && current.text == name {
					return true
				}
				stack = append(stack, current.args...)
				for _, field := range current.fields {
					stack = append(stack, field.value)
				}
				if current.left != nil {
					stack = append(stack, *current.left)
				}
				if current.right != nil {
					stack = append(stack, *current.right)
				}
			}
		}
	}
	return false
}
func runtimeFunctionsUseTextLocals(runtimeFunctions map[string]functionDecl) bool {
	for _, function := range runtimeFunctions {
		for _, kind := range function.localTypes {
			if kind == "运行时字符串" {
				return true
			}
		}
	}
	return false
}
func runtimeTextResultExpression(input value) bool {
	if input.kind != "调用" {
		return false
	}
	if input.text == "成功" && len(input.args) == 1 {
		return isRuntimeTextValue(input.args[0]) || isRuntimeArgumentText(input.args[0])
	}
	return input.text == "ifElse" && len(input.args) == 3 &&
		(runtimeTextResultExpression(input.args[1]) || runtimeTextResultExpression(input.args[2]))
}
func containsRuntimeTextVariableReference(expression value, textBuffers map[string]uint32) bool {
	if expression.kind == "引用" {
		if _, exists := textBuffers[expression.text]; exists {
			return true
		}
	}
	if expression.left != nil && containsRuntimeTextVariableReference(*expression.left, textBuffers) || expression.right != nil && containsRuntimeTextVariableReference(*expression.right, textBuffers) {
		return true
	}
	for _, argument := range expression.args {
		if containsRuntimeTextVariableReference(argument, textBuffers) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeIntegerInput(statements []statement) bool {
	for _, item := range statements {
		if containsRuntimeIntegerInput(item.value) || containsRuntimeIntegerInput(item.condition) ||
			statementsUseRuntimeIntegerInput(item.body) || statementsUseRuntimeIntegerInput(item.otherwise) {
			return true
		}
	}
	return false
}
func statementsUseRuntimeTextInput(statements []statement) bool {
	for _, item := range statements {
		if containsRuntimeTextInput(item.value) || containsRuntimeTextInput(item.condition) ||
			statementsUseRuntimeTextInput(item.body) || statementsUseRuntimeTextInput(item.otherwise) {
			return true
		}
	}
	return false
}
func containsIntegerSlotReference(expression value, slots map[string]uint32) bool {
	if expression.kind == "引用" {
		if _, exists := slots[expression.text]; exists {
			return true
		}
	}
	if expression.left != nil && containsIntegerSlotReference(*expression.left, slots) || expression.right != nil && containsIntegerSlotReference(*expression.right, slots) {
		return true
	}
	for _, argument := range expression.args {
		if containsIntegerSlotReference(argument, slots) {
			return true
		}
	}
	return false
}
func (c *windowCode) conditionalPlaceholder(condition uint32) conditionalPatch {
	patch := conditionalPatch{position: len(c.words), condition: condition}; c.words = append(c.words, 0)
	return patch
}
func (c *windowCode) patchConditional(patch conditionalPatch, target int) {
	c.words[patch.position] = branchCondition(target-patch.position, patch.condition)
}
func (c *windowCode) patchConditionFalse(patches []conditionFalsePatch, target int) {
	for _, patch := range patches {
		if patch.unconditional {
			c.words[patch.position] = branch(target - patch.position)
		} else {
			c.words[patch.position] = branchCondition(target-patch.position, patch.condition)
		}
	}
}
func (c *windowCode) unconditionalPlaceholder() int {
	position := len(c.words); c.words = append(c.words, 0)
	return position
}
func (c *windowCode) finishLoop() {
	last := len(c.loopTargets) - 1; target := c.loopTargets[last]
	for _, position := range target.breaks {
		c.words[position] = branch(len(c.words) - position)
	}
	c.loopTargets = c.loopTargets[:last]
}
func (c *windowCode) emitRuntimeFunctionBody(function functionDecl, constants map[string]constantValue) error {
	if runtimeFunctionRegisterCount(function) > 8 {
		return fmt.Errorf("运行时函数需要不超过 8 个展平参数寄存器")
	}
	previousFailurePatches := c.failurePatches; c.failurePatches = nil
	if function.body.kind == "" {
		return c.emitRuntimeStatementFunctionBody(function, constants)
	}
	parameterTypes := runtimeFunctionParameterTypes(function); parameterPayloads := runtimeFunctionParameterPayloads(function)
	parameterOffsets := runtimeFunctionParameterOffsets(parameterTypes, parameterPayloads); frameSize := uint32(alignOffset(int(runtimeFunctionParameterBytes(parameterTypes, parameterPayloads)), 16))
	returnsRuntimeText := function.runtimeReturnsResult && isTextResultPayload(function.runtimeResultPayload); textResult := resultSlot{}
	if returnsRuntimeText {
		base := frameSize; textResult = resultSlot{tag: base, value: base + 8, pointer: base + 16, length: base + 24, text: true, runtimeText: true}
		c.temporaryBufferOffset = base + 32; frameSize = uint32(alignOffset(int(base)+32+360, 16))
	}
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(29, 31, 0), storeIntegerSlot(30, 31, 8), storeIntegerSlot(19, 31, 16), addImmediate(29, 31, 0))
	if frameSize != 0 {
		c.words = append(c.words, subImmediate(31, 31, frameSize))
	}
	c.words = append(c.words, addImmediate(19, 31, 0)); c.textBuffers = map[string]uint32{}
	c.textLengths = map[string]uint32{}; c.textCapacities = map[string]uint32{}
	environment := copyConstantEnvironment(constants); slots := map[string]uint32{}
	resultSlots := map[string]resultSlot{}
	for index, parameter := range function.parameters {
		register := registerIndexForParameter(parameterTypes, runtimeFunctionParameterPayloads(function), index)
		if register >= 8 {
			return fmt.Errorf("递归运行时函数参数超过 ARM64 整数寄存器 ABI")
		}
		offset := parameterOffsets[index]; c.words = append(c.words, storeIntegerSlot(uint32(register), 19, offset))
		switch parameterTypes[index] {
		case "运行时结果":
			resultSlots[parameter.text] = resultSlot{tag: offset, value: offset + 8}; environment[parameter.text] = constantValue{kind: "运行时结果", resultPayload: "整数"}
			if isTextResultPayload(parameterPayloads[index]) {
				resultSlots[parameter.text] = resultSlot{tag: offset, value: offset + 8, pointer: offset + 16, length: offset + 24, text: true}; environment[parameter.text] = constantValue{kind: "运行时结果", resultPayload: "运行时文本"}
				c.words = append(c.words, storeIntegerSlot(uint32(register+1), 19, offset+8), storeIntegerSlot(uint32(register+2), 19, offset+16), storeIntegerSlot(uint32(register+3), 19, offset+24))
			} else {
				c.words = append(c.words, storeIntegerSlot(uint32(register+1), 19, offset+8))
			}
		case "运行时字符串":
			slots[parameter.text] = offset; environment[parameter.text] = constantValue{kind: "运行时字符串"}
			c.words = append(c.words, storeIntegerSlot(uint32(register+1), 19, offset+8))
		default:
			slots[parameter.text] = offset; environment[parameter.text] = constantValue{kind: "运行时整数"}
		}
	}
	if returnsRuntimeText {
		c.words = append(c.words, movz(0, 0), storeIntegerSlot(0, 19, textResult.tag), storeIntegerSlot(0, 19, textResult.value), storeIntegerSlot(0, 19, textResult.pointer), storeIntegerSlot(0, 19, textResult.length))
	}
	c.resultSlots = resultSlots
	if returnsRuntimeText {
		c.resultSlots["__runtime_text_return"] = textResult
		if err := c.emitRuntimeResultValue("__runtime_text_return", function.body, environment, slots, textResult); err != nil {
			return err
		}
	} else if function.runtimeReturnsResult {
		if err := c.emitRuntimeResultRegisters(function.body, environment, slots); err != nil {
			return err
		}
	} else {
		if err := c.emitRuntimeInteger(function.body, environment, slots); err != nil {
			return err
		}
	}
	c.words = append(c.words, addImmediate(31, 29, 0), loadIntegerSlot(19, 31, 16), loadIntegerSlot(30, 31, 8), loadIntegerSlot(29, 31, 0), addImmediate(31, 31, 32), 0xd65f03c0); c.emitRuntimeFailureBlock(c.exitIndex)
	c.failurePatches = previousFailurePatches
	return nil
}
func (c *windowCode) emitRuntimeStatementFunctionBody(function functionDecl, constants map[string]constantValue) error {
	slots := map[string]uint32{}; textBuffers := map[string]uint32{}
	textLengths := map[string]uint32{}; textCapacities := map[string]uint32{}
	parameterTypes := runtimeFunctionParameterTypes(function); parameterPayloads := runtimeFunctionParameterPayloads(function)
	parameterNames := map[string]bool{}; parameterResultSlots := map[string]resultSlot{}
	hasTextLocals := false; stateCount := 0
	argumentRegister := 0
	for index, parameter := range function.parameters {
		parameterNames[parameter.text] = true
		if parameterTypes[index] == "运行时字符串" {
			offset := uint32(stateCount * 8); textBuffers[parameter.text] = offset
			textLengths[parameter.text] = offset + 8; stateCount += 2
			argumentRegister += 2
			continue
		}
		if parameterTypes[index] == "运行时结果" {
			offset := uint32(stateCount * 8); parameterResultSlots[parameter.text] = resultSlot{tag: offset, value: offset + 8}
			if isTextResultPayload(parameterPayloads[index]) {
				parameterResultSlots[parameter.text] = resultSlot{tag: offset, value: offset + 8, pointer: offset + 16, length: offset + 24, text: true}; stateCount += 4
				argumentRegister += 4
			} else {
				stateCount += 2; argumentRegister += 2
			}
			continue
		}
		slots[parameter.text] = uint32(stateCount * 8)
		stateCount++
		argumentRegister++
	}
	localNames := functionLocalNames(function.statements); localResultSlots := map[string]resultSlot{}
	for _, name := range localNames {
		if parameterNames[name] {
			return fmt.Errorf("运行时函数局部变量与参数重名：%s", name)
		}
		if runtimeStatementLocalType(function, name) == "运行时字符串" {
			hasTextLocals = true; offset := uint32(stateCount * 8)
			textBuffers[name] = offset; textLengths[name] = offset + 8
			textCapacities[name] = offset + 16; stateCount += 3
			continue
		}
		if runtimeStatementLocalType(function, name) == "运行时结果" {
			offset := uint32(stateCount * 8); payload := function.localResultPayloads[name]
			result := resultSlot{tag: offset, value: offset + 8, text: isTextResultPayload(payload), runtimeText: payload == "运行时文本"}
			if result.text {
				result.pointer = offset + 16; result.length = offset + 24
				stateCount += 4
			} else {
				stateCount += 2
			}
			localResultSlots[name] = result
			continue
		}
		slots[name] = uint32(stateCount * 8)
		stateCount++
	}
	if stateCount > 502 {
		return fmt.Errorf("运行时函数状态超过 502 个槽")
	}
	temporarySize := uint32(0)
	if statementsNeedRuntimeTextSubstringScratch(function.statements) {
		temporarySize = 360
	}
	if level := statementsNeedRuntimeTextConcatScratch(function.statements); level > 0 {
		temporarySize = runtimeTextConcatBaseScratch + runtimeTextConcatScratchStride*level
	}
	frameSize := uint32(alignOffset(stateCount*8+int(temporarySize), 16)); c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(29, 31, 0), storeIntegerSlot(30, 31, 8), storeIntegerSlot(19, 31, 16), addImmediate(29, 31, 0))
	if frameSize != 0 {
		c.words = append(c.words, subImmediate(31, 31, frameSize))
	}
	c.words = append(c.words, addImmediate(19, 31, 0)); argumentRegister = 0
	for index, parameter := range function.parameters {
		if parameterTypes[index] == "运行时字符串" {
			c.words = append(c.words, storeIntegerSlot(uint32(argumentRegister), 19, textBuffers[parameter.text]), storeIntegerSlot(uint32(argumentRegister+1), 19, textLengths[parameter.text])); argumentRegister += 2
			continue
		}
		if result, exists := parameterResultSlots[parameter.text]; exists {
			c.words = append(c.words, storeIntegerSlot(uint32(argumentRegister), 19, result.tag), storeIntegerSlot(uint32(argumentRegister+1), 19, result.value))
			if result.text {
				c.words = append(c.words, storeIntegerSlot(uint32(argumentRegister+2), 19, result.pointer), storeIntegerSlot(uint32(argumentRegister+3), 19, result.length)); argumentRegister += 4
			} else {
				argumentRegister += 2
			}
			continue
		}
		c.words = append(c.words, storeIntegerSlot(uint32(argumentRegister), 19, slots[parameter.text]))
		argumentRegister++
	}
	if argumentRegister > 8 {
		return fmt.Errorf("运行时函数参数超过 ARM64 展平寄存器 ABI")
	}
	previousResultSlots := c.resultSlots; previousTextBuffers := c.textBuffers
	previousTextLengths := c.textLengths; previousTextCapacities := c.textCapacities
	previousTemporaryBufferOffset := c.temporaryBufferOffset; previousFailurePatches := c.failurePatches
	c.failurePatches = nil; c.temporaryBufferOffset = uint32(stateCount * 8)
	c.resultSlots = map[string]resultSlot{}
	for name, result := range parameterResultSlots {
		c.resultSlots[name] = result
	}
	for name, result := range localResultSlots {
		c.resultSlots[name] = result
	}
	c.textBuffers = textBuffers; c.textLengths = textLengths
	c.textCapacities = textCapacities
	if hasTextLocals {
		c.words = append(c.words, movz(0, 0))
		for _, name := range localNames {
			if _, isText := textCapacities[name]; isText {
				c.words = append(c.words, storeIntegerSlot(0, 19, textBuffers[name]), storeIntegerSlot(0, 19, textLengths[name]), storeIntegerSlot(0, 19, textCapacities[name]))
			}
		}
	}
	if len(localResultSlots) > 0 {
		c.words = append(c.words, movz(0, 0))
		for _, name := range localNames {
			result, ok := localResultSlots[name]
			if !ok {
				continue
			}
			c.words = append(c.words, storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(0, 19, result.value))
			if result.text {
				c.words = append(c.words, storeIntegerSlot(0, 19, result.pointer), storeIntegerSlot(0, 19, result.length))
			}
		}
	}
	returns := []int{}; statements := function.statements
	if function.runtimeReturnsText {
		statements = statements[:len(statements)-1]
	}
	if err := c.emitRuntimeFunctionStatements(function, statements, constants, slots, &returns); err != nil {
		return err
	}
	if len(returns) == 0 && !function.runtimeReturnsText {
		return fmt.Errorf("运行时函数语句体没有返回")
	}
	for _, position := range returns {
		c.words[position] = branch(len(c.words) - position)
	}
	if err := c.emitRuntimeTextLocalCleanup(function); err != nil {
		return err
	}
	if function.runtimeReturnsText {
		name := function.runtimeReturnLocal; c.words = append(c.words, loadIntegerSlot(1, 19, textBuffers[name]), loadIntegerSlot(2, 19, textLengths[name]), loadIntegerSlot(3, 19, textCapacities[name]), movz(0, 0))
	}
	c.words = append(c.words, addImmediate(31, 29, 0), loadIntegerSlot(19, 31, 16), loadIntegerSlot(30, 31, 8), loadIntegerSlot(29, 31, 0), addImmediate(31, 31, 32), 0xd65f03c0); c.emitRuntimeFailureBlock(c.exitIndex)
	c.failurePatches = previousFailurePatches; c.resultSlots = previousResultSlots
	c.textBuffers = previousTextBuffers; c.textLengths = previousTextLengths
	c.textCapacities = previousTextCapacities; c.temporaryBufferOffset = previousTemporaryBufferOffset
	return nil
}
func (c *windowCode) emitRuntimeTextLocalCleanup(function functionDecl) error {
	names := []string{}
	for _, name := range functionLocalNames(function.statements) {
		if name != function.runtimeReturnLocal && runtimeStatementLocalType(function, name) == "运行时字符串" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(0, 31, 0))
	for _, name := range names {
		pointer, pointerExists := c.textBuffers[name]; capacity, capacityExists := c.textCapacities[name]
		if !pointerExists || !capacityExists {
			return fmt.Errorf("无法清理运行时文本局部变量：%s", name)
		}
		c.words = append(c.words, loadIntegerSlot(1, 19, pointer), compareImmediate(1, 0)); uninitialized := c.conditionalPlaceholder(0)
		c.words = append(c.words, loadIntegerSlot(0, 19, pointer), loadIntegerSlot(1, 19, capacity)); c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0)); c.branchToRuntimeFailure(1)
		c.patchConditional(uninitialized, len(c.words))
	}
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), addImmediate(31, 31, 16))
	return nil
}
func (c *windowCode) emitResultValueRegisters(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if input.kind == "引用" {
		if result, exists := c.resultSlots[input.text]; exists {
			c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value))
			if result.text {
				c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), loadIntegerSlot(3, 19, result.length))
			} else {
				c.words = append(c.words, movz(2, 0), movz(3, 0))
			}
			return nil
		}
	}
	if runtimeResultPayload(input, constants) == "运行时文本" && runtimeResultOperandShape(input) {
		base := c.temporaryBufferOffset; result := resultSlot{tag: base, value: base + 8, pointer: base + 16, length: base + 24, text: true, runtimeText: true}
		c.words = append(c.words, movz(0, 0), storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(0, 19, result.value), storeIntegerSlot(0, 19, result.pointer), storeIntegerSlot(0, 19, result.length)); previousTemporary := c.temporaryBufferOffset
		c.temporaryBufferOffset = base + 32; err := c.emitRuntimeResultValue("__direct_text_result", input, constants, slots, result)
		c.temporaryBufferOffset = previousTemporary
		if err != nil {
			return err
		}
		c.words = append(c.words, storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(1, 19, result.value), storeIntegerSlot(2, 19, result.pointer), storeIntegerSlot(3, 19, result.length)); c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value), loadIntegerSlot(2, 19, result.pointer), loadIntegerSlot(3, 19, result.length))
		return nil
	}
	if runtimeResultOperandShape(input) {
		if err := c.emitRuntimeResultRegisters(input, constants, slots); err != nil {
			return err
		}
		if !isTextResultPayload(runtimeResultPayload(input, constants)) {
			c.words = append(c.words, movz(2, 0), movz(3, 0))
		}
		return nil
	}
	return fmt.Errorf("运行时结果函数返回必须是结果局部或成功/失败调用")
}
func (c *windowCode) emitResultFailureFlag(releaseText bool) {
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(0, 31, 0), storeIntegerSlot(1, 31, 8), storeIntegerSlot(2, 31, 16))
	if releaseText {
		c.words = append(c.words, compareImmediate(0, 1)); keep := c.conditionalPlaceholder(1)
		c.words = append(c.words, loadIntegerSlot(0, 31, 16), loadIntegerSlot(1, 31, 8)); c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0)); c.branchToRuntimeFailure(1)
		c.patchConditional(keep, len(c.words))
	}
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), compareImmediate(0, 1)); failed := c.conditionalPlaceholder(0)
	c.movImmediate(0, 0); done := c.unconditionalPlaceholder()
	c.patchConditional(failed, len(c.words)); c.movImmediate(0, 1)
	c.words[done] = branch(len(c.words) - done); c.words = append(c.words, addImmediate(31, 31, 32))
}
func (c *windowCode) emitResultErrorCode(releaseText bool) {
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(0, 31, 0), storeIntegerSlot(1, 31, 8), storeIntegerSlot(2, 31, 16))
	if releaseText {
		c.words = append(c.words, compareImmediate(0, 1)); keep := c.conditionalPlaceholder(1)
		c.words = append(c.words, loadIntegerSlot(0, 31, 16), loadIntegerSlot(1, 31, 8)); c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0)); c.branchToRuntimeFailure(1)
		c.patchConditional(keep, len(c.words))
	}
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), loadIntegerSlot(1, 31, 8), compareImmediate(0, 0)); success := c.conditionalPlaceholder(0)
	c.movRegister(0, 1); done := c.unconditionalPlaceholder()
	c.patchConditional(success, len(c.words)); c.movImmediate(0, 0)
	c.words[done] = branch(len(c.words) - done); c.words = append(c.words, addImmediate(31, 31, 32))
}
func (c *windowCode) emitRuntimeFunctionStatements(function functionDecl, statements []statement, constants map[string]constantValue, slots map[string]uint32, returns *[]int) error {
	for _, item := range statements {
		switch item.kind {
		case "变量", "赋值":
			if _, isTextLocal := c.textBuffers[item.name.text]; isTextLocal {
				if err := c.emitRuntimeTextVariable(item.name.text, item.value, constants, slots, item.kind == "变量"); err != nil {
					return err
				}
				continue
			}
			if result, exists := c.resultSlots[item.name.text]; exists {
				if err := c.emitRuntimeResultValue(item.name.text, item.value, constants, slots, result); err != nil {
					return err
				}
				c.words = append(c.words, storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(1, 19, result.value))
				if result.text {
					c.words = append(c.words, storeIntegerSlot(2, 19, result.pointer), storeIntegerSlot(3, 19, result.length))
				}
				continue
			}
			if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, storeIntegerSlot(0, 19, slots[item.name.text]))
		case "若":
			falsePatches, err := c.emitConditionFalseBranches(item.condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitRuntimeFunctionStatements(function, item.body, constants, slots, returns); err != nil {
				return err
			}
			if len(item.otherwise) == 0 {
				c.patchConditionFalse(falsePatches, len(c.words))
				continue
			}
			end := c.unconditionalPlaceholder(); c.patchConditionFalse(falsePatches, len(c.words))
			if err := c.emitRuntimeFunctionStatements(function, item.otherwise, constants, slots, returns); err != nil {
				return err
			}
			c.words[end] = branch(len(c.words) - end)
		case "凡":
			loopStart := len(c.words); c.loopTargets = append(c.loopTargets, loopTarget{continueAt: loopStart})
			falsePatches, err := c.emitConditionFalseBranches(item.condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitRuntimeLoopStatements(function, item.body, constants, slots, returns); err != nil {
				return err
			}
			c.words = append(c.words, branch(loopStart-len(c.words))); c.patchConditionFalse(falsePatches, len(c.words))
			c.finishLoop()
		case "返回":
			if function.runtimeReturnsResult {
				if err := c.emitResultValueRegisters(item.value, constants, slots); err != nil {
					return err
				}
			} else if item.value.kind == "引用" {
				if result, exists := c.resultSlots[item.value.text]; exists {
					c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value))
				} else if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
					return err
				}
			} else if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
				return err
			}
			position := c.unconditionalPlaceholder(); *returns = append(*returns, position)
		default:
			return fmt.Errorf("不支持的运行时函数语句：%s", item.kind)
		}
	}
	return nil
}
func (c *windowCode) emitRuntimeLoopStatements(function functionDecl, statements []statement, constants map[string]constantValue, slots map[string]uint32, returns *[]int) error {
	for _, item := range statements {
		switch item.kind {
		case "变量", "赋值":
			if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, storeIntegerSlot(0, 19, slots[item.name.text]))
		case "跳出":
			if len(c.loopTargets) == 0 {
				return fmt.Errorf("运行时跳出语句没有所在循环")
			}
			position := c.unconditionalPlaceholder(); last := len(c.loopTargets) - 1
			c.loopTargets[last].breaks = append(c.loopTargets[last].breaks, position)
		case "继续":
			if len(c.loopTargets) == 0 {
				return fmt.Errorf("运行时继续语句没有所在循环")
			}
			target := c.loopTargets[len(c.loopTargets)-1].continueAt; c.words = append(c.words, branch(target-len(c.words)))
		case "返回":
			if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
				return err
			}
			position := c.unconditionalPlaceholder(); *returns = append(*returns, position)
		case "若":
			falsePatches, err := c.emitConditionFalseBranches(item.condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitRuntimeLoopStatements(function, item.body, constants, slots, returns); err != nil {
				return err
			}
			if len(item.otherwise) == 0 {
				c.patchConditionFalse(falsePatches, len(c.words))
				continue
			}
			end := c.unconditionalPlaceholder(); c.patchConditionFalse(falsePatches, len(c.words))
			if err := c.emitRuntimeLoopStatements(function, item.otherwise, constants, slots, returns); err != nil {
				return err
			}
			c.words[end] = branch(len(c.words) - end)
		case "凡":
			loopStart := len(c.words); c.loopTargets = append(c.loopTargets, loopTarget{continueAt: loopStart})
			falsePatches, err := c.emitConditionFalseBranches(item.condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitRuntimeLoopStatements(function, item.body, constants, slots, returns); err != nil {
				return err
			}
			c.words = append(c.words, branch(loopStart-len(c.words))); c.patchConditionFalse(falsePatches, len(c.words))
			c.finishLoop()
		default:
			return fmt.Errorf("不支持的运行时循环体语句：%s", item.kind)
		}
	}
	return nil
}
func (c *windowCode) emitRuntimeResultRegisters(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if input.kind == "引用" {
		if result, exists := c.resultSlots[input.text]; exists && !result.text {
			c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value), movz(2, 0), movz(3, 0))
			return nil
		}
	}
	if input.kind == "调用" {
		if _, exists := c.runtimeFunctions[input.text]; exists {
			if err := c.emitRuntimeFunctionCall(input, constants, slots); err != nil {
				return err
			}
			return nil
		}
		if input.text == "ifElse" && len(input.args) == 3 {
			falsePatches, err := c.emitConditionFalseBranches(input.args[0], constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitRuntimeResultRegisters(input.args[1], constants, slots); err != nil {
				return err
			}
			end := c.unconditionalPlaceholder(); c.patchConditionFalse(falsePatches, len(c.words))
			if err := c.emitRuntimeResultRegisters(input.args[2], constants, slots); err != nil {
				return err
			}
			c.words[end] = branch(len(c.words) - end)
			return nil
		}
		if input.text == "是否失败" && len(input.args) == 1 && (input.args[0].kind == "引用" || runtimeResultOperandShape(input.args[0])) {
			if input.args[0].kind == "引用" {
				source, exists := c.resultSlots[input.args[0].text]
				if !exists {
					return fmt.Errorf("无法解析结果参数：%s", input.args[0].text)
				}
				c.words = append(c.words, loadIntegerSlot(0, 19, source.tag), compareImmediate(0, 1))
			} else if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
				return err
			} else {
				c.words = append(c.words, compareImmediate(0, 1))
			}
			success := c.conditionalPlaceholder(1); c.movImmediate(0, 1)
			c.patchConditional(success, len(c.words)); c.words = append(c.words, movz(1, 0), movz(2, 0), movz(3, 0))
			return nil
		}
		if input.text == "错误码" && len(input.args) == 1 && (input.args[0].kind == "引用" || runtimeResultOperandShape(input.args[0])) {
			if input.args[0].kind == "引用" {
				source, exists := c.resultSlots[input.args[0].text]
				if !exists {
					return fmt.Errorf("无法解析结果参数：%s", input.args[0].text)
				}
				c.words = append(c.words, loadIntegerSlot(0, 19, source.value), loadIntegerSlot(1, 19, source.tag), compareImmediate(1, 0))
			} else if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
				return err
			} else {
				c.words = append(c.words, movz(1, 0), compareImmediate(1, 0))
			}
			success := c.conditionalPlaceholder(1); c.movImmediate(0, 1)
			c.patchConditional(success, len(c.words)); c.words = append(c.words, movz(1, 0), movz(2, 0), movz(3, 0))
			return nil
		}
		if input.text == "成功" && len(input.args) == 1 {
			argument := input.args[0]
			if argument.kind == "引用" {
				if result, exists := c.resultSlots[argument.text]; exists {
					if result.text {
						return fmt.Errorf("递归运行时函数暂只支持整数载荷结果转发")
					}
					c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value), movz(2, 0), movz(3, 0))
					return nil
				}
				if err := c.emitRuntimeInteger(argument, constants, slots); err != nil {
					return err
				}
				c.words = append(c.words, 0xaa0003e1, movz(0, 0), movz(2, 0), movz(3, 0))
				return nil
			}
			if runtimeResultOperandShape(argument) {
				if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
					return err
				}
				c.words = append(c.words, movz(0, 0), movz(2, 0), movz(3, 0))
				return nil
			}
			payload := runtimeResultPayload(input, constants)
			if input.args[0].kind == "引用" {
				if constant, ok := constants[input.args[0].text]; ok && constant.kind == "运行时结果" && (constant.resultPayload == "" || constant.resultPayload == "整数") {
					payload = "整数"
				}
			}
			if payload == "整数" || payload == "运行时整数" {
				if err := c.emitRuntimeInteger(input.args[0], constants, slots); err != nil {
					return err
				}
				c.words = append(c.words, 0xaa0003e1, movz(0, 0), movz(2, 0), movz(3, 0))
				return nil
			}
			if payload == "" {
				if err := c.emitRuntimeInteger(input.args[0], constants, slots); err != nil {
					return err
				}
				c.words = append(c.words, 0xaa0003e1, movz(0, 0), movz(2, 0), movz(3, 0))
				return nil
			}
			if payload == "文本" {
				text, problem := evaluateString(input.args[0], 0, constants)
				if problem != nil {
					return fmt.Errorf("无法计算递归运行时函数的文本结果载荷：%w", problem)
				}
				label := fmt.Sprintf("runtimeFunctionResultText%d", len(c.dataLabels)); c.addData(label, append([]byte(text), '\n'), 1)
				if err := c.emitRegisterImmediate(3, int64(len(text))); err != nil {
					return err
				}
				c.address(label, 2); c.words = append(c.words, movz(0, 0), movz(1, 0))
				return nil
			}
			return fmt.Errorf("递归运行时函数暂只支持整数或编译期文本成功载荷")
		}
		if input.text == "失败" && len(input.args) == 1 {
			if err := c.emitRuntimeInteger(input.args[0], constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, compareImmediate(0, 1)); c.branchToRuntimeFailure(11)
			c.words = append(c.words, compareImmediate(0, 255)); c.branchToRuntimeFailure(8)
			c.words = append(c.words, 0xaa0003e1, movz(0, 1), movz(2, 0), movz(3, 0))
			return nil
		}
	}
	return fmt.Errorf("递归运行时结果函数需要成功、失败、ifElse 或运行时函数调用表达式")
}
func (c *windowCode) emitRuntimeFunctionCall(input value, constants map[string]constantValue, slots map[string]uint32) error {
	argumentTypes := input.runtimeArgumentTypes
	if len(argumentTypes) == 0 {
		if function, exists := c.runtimeFunctions[input.text]; exists {
			argumentTypes = runtimeFunctionParameterTypes(function)
		}
	}
	if len(argumentTypes) != len(input.args) {
		return fmt.Errorf("运行时函数调用参数类型数不匹配：%s", input.text)
	}
	argumentPayloads := input.runtimeArgumentPayloads
	if function, exists := c.runtimeFunctions[input.text]; exists {
		argumentPayloads = runtimeFunctionParameterPayloads(function)
	} else if len(argumentPayloads) != len(argumentTypes) {
		argumentPayloads = nil
	}
	registerCount := 0
	for index, argumentType := range argumentTypes {
		if argumentType == "运行时结果" && index < len(argumentPayloads) && isTextResultPayload(argumentPayloads[index]) {
			registerCount += 4
		} else if argumentType == "运行时字符串" {
			registerCount += 2
		} else if argumentType == "运行时结果" {
			registerCount += 2
		} else if argumentType == "运行时整数" {
			registerCount++
		} else {
			return fmt.Errorf("未知运行时函数参数类型：%s", argumentType)
		}
	}
	if registerCount > 8 {
		return fmt.Errorf("运行时函数调用超过 ARM64 展平寄存器 ABI")
	}
	parameterOffsets := runtimeFunctionParameterOffsets(argumentTypes, argumentPayloads); argumentBytes := runtimeFunctionParameterBytes(argumentTypes, argumentPayloads)
	if argumentBytes != 0 {
		c.words = append(c.words, subImmediate(31, 31, argumentBytes))
	}
	for index, argument := range input.args {
		offset := parameterOffsets[index]
		if argumentTypes[index] == "运行时字符串" {
			if err := c.emitRuntimeTextOperand(argument, slots, constants, 22, 21, c.temporaryBufferOffset); err != nil {
				return err
			}
			c.words = append(c.words, storeIntegerSlot(21, 31, offset), storeIntegerSlot(22, 31, offset+8))
			continue
		}
		if argumentTypes[index] == "运行时结果" {
			textPayload := index < len(argumentPayloads) && isTextResultPayload(argumentPayloads[index])
			if result, exists := c.resultSlots[argument.text]; exists {
				if result.text {
					if !textPayload {
						return fmt.Errorf("结果参数需要整数载荷结果变量：%s", argument.text)
					}
					c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value), loadIntegerSlot(8, 19, result.pointer), loadIntegerSlot(9, 19, result.length), storeIntegerSlot(0, 31, offset), storeIntegerSlot(1, 31, offset+8), storeIntegerSlot(8, 31, offset+16), storeIntegerSlot(9, 31, offset+24))
					continue
				}
				c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), loadIntegerSlot(1, 19, result.value), storeIntegerSlot(0, 31, offset), storeIntegerSlot(1, 31, offset+8))
				continue
			}
			if runtimeResultOperandShape(argument) {
				if err := c.emitResultValueRegisters(argument, constants, slots); err != nil {
					return err
				}
				c.words = append(c.words, storeIntegerSlot(0, 31, offset), storeIntegerSlot(1, 31, offset+8))
				if textPayload {
					c.words = append(c.words, storeIntegerSlot(2, 31, offset+16), storeIntegerSlot(3, 31, offset+24))
				}
				continue
			}
			return fmt.Errorf("结果参数需要整数载荷结果变量：%s", argument.text)
		}
		if err := c.emitRuntimeInteger(argument, constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, storeIntegerSlot(0, 31, offset))
	}
	registerIndex := 0
	for index, argumentType := range argumentTypes {
		offset := parameterOffsets[index]
		if argumentType == "运行时字符串" {
			c.words = append(c.words, loadIntegerSlot(uint32(registerIndex), 31, offset), loadIntegerSlot(uint32(registerIndex+1), 31, offset+8)); registerIndex += 2
			continue
		}
		if argumentType == "运行时结果" {
			if index < len(argumentPayloads) && isTextResultPayload(argumentPayloads[index]) {
				c.words = append(c.words, loadIntegerSlot(uint32(registerIndex), 31, offset), loadIntegerSlot(uint32(registerIndex+1), 31, offset+8), loadIntegerSlot(uint32(registerIndex+2), 31, offset+16), loadIntegerSlot(uint32(registerIndex+3), 31, offset+24)); registerIndex += 4
			} else {
				c.words = append(c.words, loadIntegerSlot(uint32(registerIndex), 31, offset), loadIntegerSlot(uint32(registerIndex+1), 31, offset+8)); registerIndex += 2
			}
			continue
		}
		c.words = append(c.words, loadIntegerSlot(uint32(registerIndex), 31, offset))
		registerIndex++
	}
	if argumentBytes != 0 {
		c.words = append(c.words, addImmediate(31, 31, argumentBytes))
	}
	position := len(c.words); c.words = append(c.words, 0)
	c.runtimeCallPatches = append(c.runtimeCallPatches, runtimeCallPatch{position: position, name: input.text})
	return nil
}
func (c *windowCode) patchRuntimeCalls() error {
	for _, patch := range c.runtimeCallPatches {
		target, exists := c.runtimeFunctionAt[patch.name]
		if !exists {
			return fmt.Errorf("无法解析运行时函数调用目标：%s", patch.name)
		}
		c.words[patch.position] = branchLink(target - patch.position)
	}
	return nil
}
func writeConsoleMachO(path string, expression value, constants map[string]constantValue, statements []statement, runtimeFunctions map[string]functionDecl, runtimeFunctionNames []string) error {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("当前后端只支持 macOS arm64")
	}
	image, err := makeConsoleMachO(expression, constants, statements, runtimeFunctions, runtimeFunctionNames)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, image, 0755); err != nil {
		return err
	}
	if err := os.Chmod(path, 0755); err != nil {
		return err
	}
	output, err := exec.Command("codesign", "-s", "-", "--force", "--timestamp=none", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("系统 ad-hoc 签名失败：%w：%s", err, output)
	}
	return nil
}
func makeConsoleMachO(expression value, constants map[string]constantValue, statements []statement, runtimeFunctions map[string]functionDecl, runtimeFunctionNames []string) ([]byte, error) {
	imports := []importedSymbol{{"_write", 1}, {"_exit", 1}}; usesRootInput := containsRuntimeIntegerInput(expression) || containsRuntimeTextInput(expression) || statementsUseRuntimeIntegerInput(statements) || statementsUseRuntimeTextInput(statements)
	usesInput := usesRootInput || runtimeFunctionsUseSyscall(runtimeFunctions, "readInteger") || runtimeFunctionsUseSyscall(runtimeFunctions, "readText"); usesFileRead := containsRuntimeFileRead(expression, false) || statementsUseRuntimeFileRead(statements, false) || runtimeFunctionsUseFileRead(runtimeFunctions, false)
	usesModuleRead := containsRuntimeFileRead(expression, true) || statementsUseRuntimeFileRead(statements, true) || runtimeFunctionsUseFileRead(runtimeFunctions, true); usesFileWrite := containsRuntimeFileNamed(expression, "write") || containsRuntimeFileNamed(expression, "writeFile") || statementsUseRuntimeFileNamed(statements, "write") || runtimeFunctionsUseFileNamed(runtimeFunctions, "write")
	usesArguments := containsRuntimeArguments(expression) || statementsUseRuntimeArguments(statements) || runtimeFunctionsUseArguments(runtimeFunctions); usesEnvironment := containsRuntimeEnvironmentRead(expression) || statementsUseRuntimeEnvironmentRead(statements) || runtimeFunctionsUseEnvironmentRead(runtimeFunctions); usesEnvironmentNames := containsRuntimeEnvironmentNames(expression) || statementsUseRuntimeEnvironmentNames(statements) || runtimeFunctionsUseEnvironmentNames(runtimeFunctions)
	usesMappedMemory := containsDirectRuntimeTextResultExtraction(expression, constants) || statementsUseRuntimeArray(statements) || statementsUseRuntimeMap(statements) || statementsUseRuntimeTextStorage(statements) && !statementsUseOnlyBorrowedEnvironmentText(statements) || runtimeFunctionsUseMappedMemory(runtimeFunctions) || runtimeFunctionsUseSyscall(runtimeFunctions, "readFile") || runtimeFunctionsUseSyscall(runtimeFunctions, "readModule") || runtimeFunctionsUseSyscall(runtimeFunctions, "readModuleResult")
	if usesInput || usesFileRead {
		imports = append([]importedSymbol{{"_read", 1}}, imports...)
	}
	if usesFileRead {
		imports = append(imports, importedSymbol{"_open", 1})
	}
	if usesModuleRead {
		imports = append(imports, importedSymbol{"_openat", 1}, importedSymbol{"_fcntl", 1}, importedSymbol{"_fstat", 1}, importedSymbol{"___error", 1})
	}
	if containsRuntimeFileNamed(expression, "fileSize") || statementsUseRuntimeFileNamed(statements, "fileSize") || runtimeFunctionsUseFileNamed(runtimeFunctions, "fileSize") {
		imports = append(imports, importedSymbol{"_open", 1}, importedSymbol{"_fstat", 1}, importedSymbol{"_close", 1})
	}
	if containsRuntimeFileNamed(expression, "exists") || statementsUseRuntimeFileNamed(statements, "exists") || runtimeFunctionsUseFileNamed(runtimeFunctions, "exists") {
		imports = append(imports, importedSymbol{"_open", 1}, importedSymbol{"_fstat", 1}, importedSymbol{"_close", 1})
	}
	if containsRuntimeFileNamed(expression, "isDirectory") || statementsUseRuntimeFileNamed(statements, "isDirectory") || runtimeFunctionsUseFileNamed(runtimeFunctions, "isDirectory") {
		imports = append(imports, importedSymbol{"_open", 1}, importedSymbol{"_fstat", 1}, importedSymbol{"_close", 1})
	}
	if containsRuntimeFileCreate(expression) || statementsUseRuntimeFileCreate(statements) || runtimeFileCreate(runtimeFunctions) {
		imports = append(imports, importedSymbol{"_creat", 1})
	}
	if !usesFileRead && (containsRuntimeFileAppend(expression) || statementsUseAppend(statements) || runtimeAppend(runtimeFunctions)) {
		imports = append(imports, importedSymbol{"_open", 1})
	}
	if containsRuntimeFileRemove(expression) || statementsUseRuntimeFileNamed(statements, "removeFile") || runtimeFileRemove(runtimeFunctions) {
		imports = append(imports, importedSymbol{"_unlink", 1})
	}
	if containsRuntimeFileRename(expression) || containsRuntimeFileNamed(expression, "replaceFile") || statementsUseRuntimeFileNamed(statements, "renameFile") || statementsUseRuntimeFileNamed(statements, "replaceFile") || runtimeFileRename(runtimeFunctions) || runtimeFileReplace(runtimeFunctions) {
		imports = append(imports, importedSymbol{"_rename", 1})
	}
	if containsRuntimeFileNamed(expression, "createDirectory") || statementsUseRuntimeFileNamed(statements, "createDirectory") || runtimeFunctionsUseFileNamed(runtimeFunctions, "createDirectory") {
		imports = append(imports, importedSymbol{"_mkdir", 1})
	}
	if containsRuntimeFileNamed(expression, "removeDirectory") || statementsUseRuntimeFileNamed(statements, "removeDirectory") || runtimeFunctionsUseFileNamed(runtimeFunctions, "removeDirectory") {
		imports = append(imports, importedSymbol{"_rmdir", 1})
	}
	if containsRuntimeFileNamed(expression, "setFileMode") || statementsUseRuntimeFileNamed(statements, "setFileMode") || runtimeFunctionsUseFileNamed(runtimeFunctions, "setFileMode") {
		imports = append(imports, importedSymbol{"_chmod", 1})
	}
	if containsRuntimeFileNamed(expression, "changeFileMode") || statementsUseRuntimeFileNamed(statements, "changeFileMode") || runtimeFunctionsUseFileNamed(runtimeFunctions, "changeFileMode") {
		imports = append(imports, importedSymbol{"_open", 1}, importedSymbol{"_fchmod", 1}, importedSymbol{"_close", 1})
	} else if usesFileRead || usesFileWrite {
		imports = append(imports, importedSymbol{"_close", 1})
	}
	if usesMappedMemory || usesFileRead {
		imports = append(imports, importedSymbol{"_mmap", 1}, importedSymbol{"_munmap", 1})
	}
	if usesEnvironment {
		imports = append(imports, importedSymbol{"_getenv", 1})
	}
	if containsRuntimeEnvironmentNames(expression) || statementsUseRuntimeEnvironmentNames(statements) || runtimeFunctionsUseEnvironmentNames(runtimeFunctions) {
		imports = append(imports, importedSymbol{"_environ", 1})
	}
	c := windowCode{dataLabels: map[string]int{}, codeLabels: map[string]int{}}; c.usesArguments = usesArguments
	c.usesEnvironment = usesEnvironment || usesEnvironmentNames; c.borrowedEnvironmentText = map[string]bool{}
	c.runtimeFunctions = runtimeFunctions; c.runtimeFunctionOrder = runtimeFunctionNames
	c.runtimeFunctionAt = map[string]int{}; c.writeIndex, c.exitIndex = -1, -1
	c.fileReadIndex, c.fileOpenIndex, c.fileCreateIndex, c.fileCloseIndex, c.fileRemoveIndex, c.fileRenameIndex, c.fileMkdirIndex, c.fileRmdirIndex, c.fileChmodIndex, c.fileFchmodIndex, c.moduleStatIndex, c.moduleErrorIndex = -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1; c.arrayMapIndex, c.arrayUnmapIndex = -1, -1
	c.environmentIndex = -1; c.getenvIndex = -1
	for index, item := range imports {
		switch item.name {
		case "_write":
			c.writeIndex = index
		case "_exit":
			c.exitIndex = index
		case "_read":
			c.fileReadIndex = index
		case "_open":
			c.fileOpenIndex = index
		case "_openat":
			c.moduleOpenIndex = index
		case "_fcntl":
			c.moduleControlIndex = index
		case "_fstat":
			c.moduleStatIndex = index
		case "___error":
			c.moduleErrorIndex = index
		case "_creat":
			c.fileCreateIndex = index
		case "_close":
			c.fileCloseIndex = index
		case "_unlink":
			c.fileRemoveIndex = index
		case "_rename":
			c.fileRenameIndex = index
		case "_mkdir":
			c.fileMkdirIndex = index
		case "_rmdir":
			c.fileRmdirIndex = index
		case "_chmod":
			c.fileChmodIndex = index
		case "_fchmod":
			c.fileFchmodIndex = index
		case "_mmap":
			c.arrayMapIndex = index
		case "_munmap":
			c.arrayUnmapIndex = index
		case "_getenv":
			c.getenvIndex = index
		case "_environ":
			c.environmentIndex = index
		}
	}
	if err := c.emitConsoleProgram(expression, constants, statements); err != nil {
		return nil, err
	}
	if err := c.patchRuntimeCalls(); err != nil {
		return nil, err
	}
	constOffset := alignOffset(codeFileOff+len(c.words)*4, 8); textEnd := constOffset + len(c.constants)
	if textEnd > 1<<20 {
		return nil, fmt.Errorf("控制台 Mach-O 文本段超过 1 MiB")
	}
	consoleDataOffset := alignOffset(textEnd, int(pageSize)); consoleLinkOffset := consoleDataOffset + int(pageSize)
	if err := c.resolve(uint64(constOffset), uint64(consoleDataOffset)); err != nil {
		return nil, err
	}
	bind := makeWindowBinds(imports); symbolOffset := alignOffset(consoleLinkOffset+len(bind), 8)
	stringTable := []byte{0}; stringIndices := []uint32{}
	for _, item := range imports {
		stringIndices = append(stringIndices, uint32(len(stringTable))); stringTable = append(stringTable, []byte(item.name+"\x00")...)
	}
	stringOffset := symbolOffset + 16*len(imports); fileEnd := stringOffset + len(stringTable)
	image := make([]byte, fileEnd); commandAt, commandCount := 32, 0
	writeSegment(image, commandAt, "__PAGEZERO", 72, 0, 0x100000000, 0, 0, 0, 0, 0); commandAt += 72
	textCommand := commandAt; writeSegment(image, textCommand, "__TEXT", 232, imageBase, uint64(consoleDataOffset), 0, uint64(consoleDataOffset), 5, 5, 2)
	writeSection(image, textCommand+72, "__text", "__TEXT", imageBase+uint64(codeFileOff), uint64(len(c.words)*4), uint32(codeFileOff), 2, 0x80000400); writeSection(image, textCommand+152, "__const", "__TEXT", imageBase+uint64(constOffset), uint64(len(c.constants)), uint32(constOffset), 3, 0)
	commandAt += 232; dataCommand := commandAt
	writeSegment(image, dataCommand, "__DATA", 152, imageBase+uint64(consoleDataOffset), pageSize, uint64(consoleDataOffset), pageSize, 3, 3, 1); writeSection(image, dataCommand+72, "__got", "__DATA", imageBase+uint64(consoleDataOffset), uint64(len(imports)*8), uint32(consoleDataOffset), 3, 6)
	commandAt += 152; writeSegment(image, commandAt, "__LINKEDIT", 72, imageBase+uint64(consoleLinkOffset), uint64(alignOffset(fileEnd-consoleLinkOffset, int(pageSize))), uint64(consoleLinkOffset), uint64(fileEnd-consoleLinkOffset), 1, 1, 0)
	commandAt += 72; writeDyldInfo(image, commandAt, consoleLinkOffset, len(bind))
	commandAt += 48; writeDylinker(image, commandAt)
	commandAt += 32; commandAt += writeDylibPath(image, commandAt, "/usr/lib/libSystem.B.dylib")
	writeBuildVersion(image, commandAt); commandAt += 24
	writeMain(image, commandAt, uint64(codeFileOff)); commandAt += 24
	writeUUID(image, commandAt, wordsBytes(c.words)); commandAt += 24
	writeSymbolTable(image, commandAt, symbolOffset, stringOffset, len(imports), len(stringTable)); commandAt += 24
	writeDynamicSymbolTable(image, commandAt, len(imports)); commandAt += 80
	commandCount = 12
	if commandAt > codeFileOff {
		return nil, fmt.Errorf("Mach-O 加载命令区超出代码起始偏移")
	}
	writeDynamicHeader(image, commandCount, commandAt-32); copy(image[codeFileOff:], wordsBytes(c.words))
	copy(image[constOffset:], c.constants); copy(image[consoleLinkOffset:], bind)
	for index, stringIndex := range stringIndices {
		writeNlist(image, symbolOffset+16*index, stringIndex)
	}
	copy(image[stringOffset:], stringTable)
	return image, nil
}
func (c *windowCode) emitConsoleProgram(expression value, constants map[string]constantValue, statements []statement) error {
	slots := map[string]uint32{}; setRuntimeFunctionPayload(&expression, c.runtimeFunctions)
	environment := consoleCodegenEnvironment(statements, constants, c.runtimeFunctions); c.arraySlots = map[string]arraySlot{}
	c.mapSlots = map[string]mapSlot{}; c.resultSlots = map[string]resultSlot{}
	c.slotCount = 0
	if len(c.runtimeFunctionOrder) > 0 {
		entryJump := c.unconditionalPlaceholder()
		for _, name := range c.runtimeFunctionOrder {
			previousResultSlots := c.resultSlots; previousTextBuffers := c.textBuffers
			previousTextLengths := c.textLengths; previousTextCapacities := c.textCapacities
			c.resultSlots = map[string]resultSlot{}; c.textBuffers = map[string]uint32{}
			c.textLengths = map[string]uint32{}; c.textCapacities = map[string]uint32{}
			c.runtimeFunctionAt[name] = len(c.words)
			if err := c.emitRuntimeFunctionBody(c.runtimeFunctions[name], constants); err != nil {
				return err
			}
			c.resultSlots = previousResultSlots; c.textBuffers = previousTextBuffers
			c.textLengths = previousTextLengths; c.textCapacities = previousTextCapacities
		}
		c.words[entryJump] = branch(len(c.words) - entryJump)
	}
	c.textBuffers = map[string]uint32{}; c.textLengths = map[string]uint32{}
	c.textCapacities = map[string]uint32{}; textVariables := []string{}
	borrowedTextVariables := map[string]bool{}
	for _, item := range statements {
		if item.kind == "变量" {
			if isRuntimeTextValue(item.value) || isRuntimeFileRead(item.value) || isRuntimeArgumentText(item.value) || isTextResultExtraction(item.value) || isStoredTextReturnCall(item.value) {
				textVariables = append(textVariables, item.name.text); borrowedTextVariables[item.name.text] = isRuntimeEnvironmentRead(item.value)
				c.borrowedEnvironmentText[item.name.text] = isRuntimeEnvironmentRead(item.value)
				continue
			}
			slots[item.name.text] = c.slotCount * 8
			if isRuntimeMapValue(item.value) {
				base := slots[item.name.text]; c.mapSlots[item.name.text] = mapSlot{data: base, length: base + 8, capacity: base + 16}
				c.slotCount += 3
				continue
			}
			if resultType, _ := constantType(item.value, environment); resultType == "运行时结果" {
				payload := environment[item.name.text].resultPayload; result := resultSlot{tag: slots[item.name.text], value: slots[item.name.text] + 8, text: isTextResultPayload(payload), runtimeText: payload == "运行时文本"}
				if result.text {
					result.pointer = slots[item.name.text] + 16; result.length = slots[item.name.text] + 24
					c.slotCount += 4
				} else {
					c.slotCount += 2
				}
				c.resultSlots[item.name.text] = result
				continue
			}
			if item.value.kind == "调用" && item.value.text == "array" {
				base := c.slotCount * 8; arrayType, _ := constantType(item.value, environment)
				c.arraySlots[item.name.text] = arraySlot{data: base, length: base + 8, capacity: base + 16, text: arrayType == "字符串数组" || arrayType == "运行时字符串数组"}; c.slotCount += 3
				continue
			}
			c.slotCount++
		}
	}
	for index, name := range textVariables {
		base := c.slotCount*8 + uint32(index)*24; c.textBuffers[name] = base
		c.textLengths[name] = base + 8; c.textCapacities[name] = base + 16
		if borrowedTextVariables[name] {
			c.textCapacities[name] = 0
		}
	}
	c.temporaryBufferOffset = c.slotCount*8 + uint32(len(textVariables))*24; temporarySize := uint32(64)
	if containsDirectRuntimeTextResultExtraction(expression, environment) {
		temporarySize = 392 // result descriptor plus staged text operations
	}
	if containsRuntimeEnvironmentNames(expression) || statementsUseRuntimeEnvironmentNames(statements) {
		temporarySize = 256*8 + 424 // pointer array, count/index, and enumeration state slots
	}
	if runtimeTextSubstringNeedsScratch(expression) || statementsNeedRuntimeTextSubstringScratch(statements) {
		temporarySize = 360 // 256-byte slice destination, concat scratch, byte length, and integer output
	}
	concatScratchLevel := runtimeTextConcatScratchLevel(expression)
	if statementLevel := statementsNeedRuntimeTextConcatScratch(statements); statementLevel > concatScratchLevel {
		concatScratchLevel = statementLevel
	}
	if concatScratchLevel > 0 {
		temporarySize = runtimeTextConcatBaseScratch + runtimeTextConcatScratchStride*concatScratchLevel
	}
	if c.runtimeTextComparisonNeedsScratch(expression, environment, slots) || c.statementsNeedRuntimeTextComparisonScratch(statements, environment, slots) {
		comparisonSize := runtimeTextConcatBaseScratch + runtimeTextConcatScratchStride*concatScratchLevel + runtimeTextConcatScratchStride
		if comparisonSize > temporarySize {
			temporarySize = comparisonSize
		}
	}
	if statementsUseRuntimeArray(statements) &&
		(containsRuntimeIntegerInput(expression) || containsRuntimeTextInput(expression) || statementsUseRuntimeIntegerInput(statements) || statementsUseRuntimeTextInput(statements)) && temporarySize < 320 {
		temporarySize = 320 // runtime text array operands use a 64-byte input scratch area at +256
	}
	if expression.kind == "调用" && expression.text == "textConcat" {
		temporarySize += 8 // retain the byte length when a concatenation is the root output
	}
	if c.usesArguments {
		c.argumentCountOffset = c.temporaryBufferOffset + temporarySize; c.argumentVectorOffset = c.argumentCountOffset + 8
		temporarySize += 16
	}
	frameSize := uint32(alignOffset(int(c.temporaryBufferOffset+temporarySize), 16))
	if frameSize > 32768 {
		return fmt.Errorf("控制台整数变量超过原生栈帧上限")
	}
	kind, problem := constantType(expression, environment)
	if problem != nil {
		return fmt.Errorf("无法确定控制台输出类型：%w", problem)
	}
	writeIndex, exitIndex := c.writeIndex, c.exitIndex
	if frameSize >= 4096 {
		c.words = append(c.words, subImmediate(31, 31, frameSize>>12)|1<<22)
	}
	c.words = append(c.words, subImmediate(31, 31, frameSize&4095), addImmediate(19, 31, 0)); c.emitConsoleStackAddress(20, frameSize)
	if c.usesArguments {
		c.words = append(c.words, storeIntegerSlot(0, 19, c.argumentCountOffset), storeIntegerSlot(1, 19, c.argumentVectorOffset))
	}
	if err := c.emitConsoleStatements(statements, constants, slots); err != nil {
		return err
	}
	if kind == "运行时字符串" {
		if expression.kind == "调用" && expression.text == "结果值" && len(expression.args) == 1 && expression.args[0].kind == "引用" {
			result, exists := c.resultSlots[expression.args[0].text]
			if !exists {
				return fmt.Errorf("无法解析结果变量：%s", expression.args[0].text)
			}
			if !result.text {
				c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), compareImmediate(0, 1)); failed := c.conditionalPlaceholder(1)
				c.words = append(c.words, loadIntegerSlot(0, 19, result.value)); success := c.unconditionalPlaceholder()
				c.patchConditional(failed, len(c.words)); c.call(exitIndex)
				c.words[success] = branch(len(c.words) - success)
				return nil
			}
			c.words = append(c.words, loadIntegerSlot(2, 19, result.tag), compareImmediate(2, 1)); success := c.conditionalPlaceholder(1)
			c.words = append(c.words, loadIntegerSlot(0, 19, result.value)); c.call(exitIndex)
			c.patchConditional(success, len(c.words)); c.words = append(c.words, loadIntegerSlot(1, 19, result.pointer), loadIntegerSlot(2, 19, result.length))
			c.words = append(c.words, addImmediate(2, 2, 1)); c.movImmediate(0, 1)
			c.call(writeIndex); c.movImmediate(0, 0)
			c.call(exitIndex)
		} else if isDirectTextResultExtraction(expression) {
			if err := c.emitRuntimeResultTextOutput(expression, constants, slots, writeIndex, exitIndex); err != nil {
				return err
			}
		} else if isRuntimeTextInput(expression) {
			if err := c.emitReadText(c.temporaryBufferOffset, nil, false, 0, writeIndex, exitIndex); err != nil {
				return err
			}
		} else if expression.kind == "引用" {
			buffer, bufferExists := c.textBuffers[expression.text]; length, lengthExists := c.textLengths[expression.text]
			if !bufferExists || !lengthExists {
				return fmt.Errorf("无法解析运行时文本变量：%s", expression.text)
			}
			if borrowed, exists := c.borrowedEnvironmentText[expression.text]; exists && borrowed {
				c.words = append(c.words, loadIntegerSlot(1, 19, buffer), loadIntegerSlot(2, 19, length)); c.emitRuntimeEnvironmentTextOutputRegisters(1, 2, writeIndex, exitIndex)
				c.emitRuntimeFailureBlock(exitIndex)
				return nil
			}
			c.emitRuntimeTextOutputPointer(buffer, length, writeIndex, exitIndex)
		} else if isRuntimeArgumentText(expression) {
			if err := c.emitRuntimeArgumentTextOperand(expression, slots, constants, 22, 21); err != nil {
				return err
			}
			if err := c.emitRuntimeArgumentTextOutput(writeIndex); err != nil {
				return err
			}
		} else if isRuntimeFileRead(expression) {
			if err := c.emitRuntimeFileOutput(expression, constants, slots, writeIndex, exitIndex); err != nil {
				return err
			}
		} else if expression.kind == "调用" && expression.text == "substring" {
			length, err := c.emitRuntimeTextSubstring(expression, c.temporaryBufferOffset, constants, slots, true)
			if err != nil {
				return err
			}
			c.emitRuntimeTextOutputRegister(c.temporaryBufferOffset, length, writeIndex, exitIndex)
		} else if expression.kind == "调用" && expression.text == "textConcat" {
			lengthOffset := c.temporaryBufferOffset + 64
			if _, err := c.emitRuntimeTextConcat(expression, c.temporaryBufferOffset, &lengthOffset, slots, constants); err != nil {
				return err
			}
			c.emitRuntimeTextOutput(c.temporaryBufferOffset, lengthOffset, writeIndex, exitIndex)
		} else if expression.kind == "调用" && expression.runtimeFunctionText {
			if err := c.emitRuntimeTextFunctionOutput(expression, constants, slots, writeIndex, exitIndex); err != nil {
				return err
			}
		} else if expression.kind == "调用" && expression.text == "arrayGet" {
			if err := c.emitRuntimeTextOperand(expression, slots, constants, 22, 21, c.temporaryBufferOffset+256); err != nil {
				return err
			}
			c.emitRuntimeTextOutputRegisters(21, 22, writeIndex, exitIndex)
		} else if isRuntimeEnvironmentNames(expression) {
			if err := c.emitRuntimeEnvironmentNamesTextOutput(expression, constants, slots, writeIndex, exitIndex); err != nil {
				return err
			}
			c.emitRuntimeFailureBlock(exitIndex)
			return nil
		} else if isRuntimeEnvironmentRead(expression) {
			buffer, length, err := c.emitRuntimeEnvironmentRead(expression, constants, slots)
			if err != nil {
				return err
			}
			c.emitRuntimeEnvironmentTextOutputRegisters(buffer, length, writeIndex, exitIndex)
		} else {
			return fmt.Errorf("原生运行时文本只支持 readText（）、有界文本变量、substring 或 textConcat")
		}
		c.emitRuntimeFailureBlock(exitIndex)
		return nil
	}
	if kind == "字符串" {
		text, problem := evaluateString(expression, 0, constants)
		if problem != nil {
			return fmt.Errorf("无法计算控制台输出字符串：%w", problem)
		}
		if err := c.emitConsoleString(text, writeIndex, exitIndex); err != nil {
			return err
		}
		c.emitRuntimeFailureBlock(exitIndex)
		return nil
	}
	if kind == "布尔值" || kind == "运行时布尔值" {
		expression = value{kind: "调用", text: "ifElse", args: []value{expression, {kind: "整数", text: "1"}, {kind: "整数", text: "0"}}}; kind = "运行时整数"
	} else if kind != "整数" && kind != "运行时整数" {
		return fmt.Errorf("不支持的控制台输出类型：%s", kind)
	}
	c.movImmediate(25, '\n'); c.words = append(c.words, subImmediate(20, 20, 1), 0x39000299)
	if err := c.emitRuntimeInteger(expression, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, 0xf100001f) // cmp x0, #0
	signBranch := len(c.words); c.words = append(c.words, 0)
	c.movImmediate(22, 1) // remember a negative input until after its digits
	c.words = append(c.words, 0xcb0003f5); positiveBranch := len(c.words)
	c.words = append(c.words, 0); positiveIndex := len(c.words)
	c.words = append(c.words, 0xaa0003f5) // mov x21, x0
	c.movImmediate(22, 0); conversionIndex := len(c.words)
	c.words[signBranch] = branchCondition(positiveIndex-signBranch, 10) // b.ge
	c.words[positiveBranch] = branch(conversionIndex - positiveBranch); c.movImmediate(23, 10)
	digitLoop := len(c.words); c.words = append(c.words, 0x9ad70ab8, 0x9b17d719, addWordImmediate(25, 25, 48))
	c.words = append(c.words, subImmediate(20, 20, 1), 0x39000299, 0xaa1803f5); c.words = append(c.words, 0xf10002bf)
	loopBranch := len(c.words)
	c.words = append(c.words, branchCondition(digitLoop-loopBranch, 1)) // b.ne
	c.words = append(c.words, 0xf10002df)                               // cmp x22, #0
	minusBranch := len(c.words); c.words = append(c.words, 0)
	c.words = append(c.words, subImmediate(20, 20, 1)); c.movImmediate(25, '-')
	c.words = append(c.words, 0x39000299); noMinusIndex := len(c.words)
	c.words[minusBranch] = branchCondition(noMinusIndex-minusBranch, 0) // b.eq
	c.movImmediate(0, 1)                                                // stdout
	c.movRegister(1, 20); c.emitConsoleStackAddress(2, frameSize)
	c.words = append(c.words, 0xcb140042)
	c.call(writeIndex) // write(stdout, buffer, length)
	c.movImmediate(0, 0)
	c.call(exitIndex) // _exit(0)
	c.emitRuntimeFailureBlock(exitIndex)
	return nil
}
func consoleCodegenEnvironment(statements []statement, constants map[string]constantValue, runtimeFunctions map[string]functionDecl) map[string]constantValue {
	environment := copyConstantEnvironment(constants)
	for _, function := range runtimeFunctions {
		parameterTypes := runtimeFunctionParameterTypes(function)
		for index, parameter := range function.parameters {
			payload := ""
			if parameterTypes[index] == "运行时结果" {
				payload = "整数"
			}
			environment[parameter.text] = constantValue{kind: parameterTypes[index], resultPayload: payload}
		}
		for _, name := range functionLocalNames(function.statements) {
			if runtimeStatementLocalType(function, name) != "运行时结果" {
				continue
			}
			for _, item := range function.statements {
				if item.kind == "变量" && item.name.text == name {
					environment[name] = constantValue{kind: "运行时结果", resultPayload: runtimeResultPayload(item.value, environment)}
					break
				}
			}
		}
	}
	for _, item := range statements {
		switch item.kind {
		case "变量":
			if isRuntimeMapValue(item.value) {
				environment[item.name.text] = constantValue{kind: "整数映射"}
				continue
			}
			if isRuntimeTextValue(item.value) || isRuntimeFileRead(item.value) || isRuntimeArgumentText(item.value) || isTextResultExtraction(item.value) || isStoredTextReturnCall(item.value) {
				environment[item.name.text] = constantValue{kind: "运行时字符串"}
				continue
			}
			if item.value.kind == "调用" && item.value.text == "array" {
				arrayType, problem := constantType(item.value, environment)
				if problem == nil && (arrayType == "字符串数组" || arrayType == "运行时字符串数组") {
					items, _ := runtimeTextArrayItems(item.value, environment); environment[item.name.text] = constantValue{kind: "运行时字符串数组", array: items}
				} else {
					environment[item.name.text] = constantValue{kind: "运行时" + strings.TrimPrefix(strings.TrimSuffix(arrayType, "数组"), "运行时") + "数组", array: make([]constantValue, len(item.value.args))}
				}
				continue
			}
			if kind, problem := constantType(item.value, environment); problem == nil && kind == "运行时结果" {
				environment[item.name.text] = constantValue{kind: "运行时结果", resultPayload: runtimeResultPayload(item.value, environment)}
				continue
			}
			environment[item.name.text] = consoleCodegenValue(item.value, environment)
		case "赋值":
			if item.value.kind == "调用" && item.value.text == "映射置" {
				continue
			}
			if item.value.kind == "调用" && (item.value.text == "arraySet" || item.value.text == "arrayPush") {
				continue
			}
			if kind, problem := constantType(item.value, environment); problem == nil && kind == "运行时结果" {
				payload := environment[item.name.text].resultPayload
				if incoming := runtimeResultPayload(item.value, environment); incoming != "" {
					payload = incoming
				}
				environment[item.name.text] = constantValue{kind: "运行时结果", resultPayload: payload}
				continue
			}
			if isRuntimeTextValue(item.value) || isRuntimeFileRead(item.value) || isRuntimeArgumentText(item.value) || isTextResultExtraction(item.value) || isStoredTextReturnCall(item.value) {
				environment[item.name.text] = constantValue{kind: "运行时字符串"}
				continue
			}
			environment[item.name.text] = consoleCodegenValue(item.value, environment)
		case "若", "凡":
			markConsoleVariablesRuntime(assignedConsoleVariables(item.body), environment); markConsoleVariablesRuntime(assignedConsoleVariables(item.otherwise), environment)
		}
	}
	return environment
}
func consoleCodegenValue(input value, environment map[string]constantValue) constantValue {
	kind, problem := constantType(input, environment)
	if problem == nil && kind == "运行时结果" {
		return constantValue{kind: "运行时结果"}
	}
	if problem == nil && (kind == "布尔值" || kind == "运行时布尔值") {
		return constantValue{kind: "运行时布尔值"}
	}
	integer, problem := integerValue(input, environment)
	if problem != nil || kind == "运行时整数" {
		return constantValue{kind: "运行时整数"}
	}
	return constantValue{kind: "整数", integer: integer}
}
func (c *windowCode) emitRuntimeResultValue(name string, input value, constants map[string]constantValue, slots map[string]uint32, result resultSlot) error {
	if isModuleReadResult(input) {
		return c.emitModuleReadResult(input, constants, slots, result)
	}
	if input.kind == "引用" {
		source, exists := c.resultSlots[input.text]
		if !exists {
			return fmt.Errorf("无法解析结果变量：%s", input.text)
		}
		if result.text {
			if !source.text {
				return fmt.Errorf("无法把整数结果复制到文本结果变量")
			}
			if source.runtimeText {
				c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), compareImmediate(2, 0)); bufferReady := c.conditionalPlaceholder(1)
				if err := c.emitRegisterImmediate(0, 16*1024*1024); err != nil {
					return err
				}
				if err := c.emitArrayMapCall(0); err != nil {
					return err
				}
				c.words = append(c.words, storeIntegerSlot(0, 19, result.pointer), movz(0, 0), storeIntegerSlot(0, 19, result.length)); c.patchConditional(bufferReady, len(c.words))
				c.words = append(c.words, loadIntegerSlot(3, 19, source.length), loadIntegerSlot(5, 19, source.pointer), loadIntegerSlot(2, 19, result.pointer), movz(6, 0)); copyLoop := len(c.words)
				c.words = append(c.words, compareRegister(6, 3)); copyDone := c.conditionalPlaceholder(10)
				c.words = append(c.words, addRegister(7, 5, 6), loadByte(4, 7), addRegister(7, 2, 6), storeByte(4, 7), addImmediate(6, 6, 1)); c.words = append(c.words, branch(copyLoop-len(c.words)))
				c.patchConditional(copyDone, len(c.words)); c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), addRegister(7, 2, 3), movz(4, 10), storeByte(4, 7))
				c.words = append(c.words, loadIntegerSlot(0, 19, source.tag), loadIntegerSlot(1, 19, source.value), loadIntegerSlot(2, 19, result.pointer), loadIntegerSlot(3, 19, source.length))
				return nil
			}
		}
		c.words = append(c.words, loadIntegerSlot(0, 19, source.tag), loadIntegerSlot(1, 19, source.value))
		if result.text {
			c.words = append(c.words, loadIntegerSlot(2, 19, source.pointer), loadIntegerSlot(3, 19, source.length))
		} else if source.text {
			return fmt.Errorf("无法把文本结果复制到整数结果变量")
		} else {
			c.movImmediate(2, 0); c.movImmediate(3, 0)
		}
		return nil
	}
	if input.kind == "调用" && input.runtimeFunction && input.runtimeFunctionResult {
		textPayload := isTextResultPayload(input.runtimeFunctionPayload)
		if result.text != textPayload {
			return fmt.Errorf("运行时结果函数载荷与目标结果变量类型不匹配")
		}
		if err := c.emitRuntimeFunctionCall(input, constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(1, 19, result.value))
		if result.text {
			c.words = append(c.words, storeIntegerSlot(2, 19, result.pointer), storeIntegerSlot(3, 19, result.length))
		} else {
			c.words = append(c.words, movz(2, 0), movz(3, 0))
		}
		return nil
	}
	if input.kind == "调用" && input.text == "ifElse" && len(input.args) == 3 {
		falsePatches, err := c.emitConditionFalseBranches(input.args[0], constants, slots)
		if err != nil {
			return err
		}
		if err := c.emitRuntimeResultValue(name, input.args[1], constants, slots, result); err != nil {
			return err
		}
		end := c.unconditionalPlaceholder(); c.patchConditionFalse(falsePatches, len(c.words))
		if err := c.emitRuntimeResultValue(name, input.args[2], constants, slots, result); err != nil {
			return err
		}
		c.words[end] = branch(len(c.words) - end)
		return nil
	}
	if input.kind != "调用" || len(input.args) != 1 || input.text != "成功" && input.text != "失败" {
		return fmt.Errorf("原生结果只支持成功、失败、ifElse 或结果变量")
	}
	if input.text == "成功" && runtimeResultPayload(input, constants) == "运行时文本" {
		if !result.runtimeText {
			return fmt.Errorf("运行时文本载荷要求运行时文本结果变量")
		}
		if err := c.emitRuntimeResultTextBuffer(result); err != nil {
			return err
		}
		switch {
		case isRuntimeTextInput(input.args[0]):
			if err := c.emitReadDynamicText(result.pointer, result.length, result.value, 0, c.exitIndex); err != nil {
				return err
			}
		case input.args[0].kind == "调用" && input.args[0].text == "textConcat":
			if err := c.emitRuntimeResultTextConcat(input.args[0], result, slots, constants); err != nil {
				return err
			}
		case input.args[0].kind == "调用" && input.args[0].text == "arrayGet":
			if err := c.emitRuntimeResultTextArrayElement(input.args[0], result, slots, constants); err != nil {
				return err
			}
		case input.args[0].kind == "调用" && input.args[0].text == "substring":
			if err := c.emitRuntimeResultTextSubstring(input.args[0], result, slots, constants); err != nil {
				return err
			}
		case isRuntimeArgumentText(input.args[0]):
			if err := c.emitRuntimeResultTextArgument(input.args[0], result, slots, constants); err != nil {
				return err
			}
		default:
			return fmt.Errorf("无法生成运行时文本结果载荷")
		}
		c.words = append(c.words, movz(0, 0), loadIntegerSlot(1, 19, result.value), loadIntegerSlot(2, 19, result.pointer), loadIntegerSlot(3, 19, result.length))
		return nil
	}
	if input.text == "成功" && runtimeResultPayload(input, constants) == "文本" {
		if !result.text {
			return fmt.Errorf("无法把文本载荷写入整数结果变量")
		}
		text, problem := evaluateString(input.args[0], 0, constants)
		if problem != nil {
			return fmt.Errorf("无法计算文本结果载荷：%w", problem)
		}
		label := fmt.Sprintf("runtimeResultText%d", len(c.dataLabels)); data := append([]byte(text), '\n')
		c.addData(label, data, 1); c.movImmediate(0, 0)
		c.movImmediate(1, 0)
		if err := c.emitRegisterImmediate(3, int64(len(text))); err != nil {
			return err
		}
		c.address(label, 2)
		return nil
	}
	if result.text && input.text == "成功" {
		return fmt.Errorf("文本结果变量要求带编译期文本载荷的成功结果")
	}
	if err := c.emitRuntimeInteger(input.args[0], constants, slots); err != nil {
		return err
	}
	c.movRegister(1, 0); c.movImmediate(0, 0)
	if input.text == "失败" {
		c.words = append(c.words, compareImmediate(1, 1)); c.branchToRuntimeFailure(11)
		c.words = append(c.words, compareImmediate(1, 255)); c.branchToRuntimeFailure(8)
		c.movImmediate(0, 1)
	}
	if result.runtimeText {
		c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), movz(3, 0))
	} else {
		c.movImmediate(2, 0); c.movImmediate(3, 0)
	}
	return nil
}
func (c *windowCode) emitRuntimeResultTextBuffer(result resultSlot) error {
	c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), compareImmediate(2, 0))
	bufferReady := c.conditionalPlaceholder(1) // b.ne when the owned buffer already exists
	if err := c.emitRegisterImmediate(0, 16*1024*1024); err != nil {
		return err
	}
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, result.pointer), movz(0, 0), storeIntegerSlot(0, 19, result.length)); c.patchConditional(bufferReady, len(c.words))
	if err := c.emitRegisterImmediate(1, 16*1024*1024); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(1, 19, result.value))
	return nil
}
func (c *windowCode) emitRuntimeResultTextConcat(input value, result resultSlot, slots map[string]uint32, constants map[string]constantValue) error {
	staged := c.temporaryBufferOffset
	if err := c.emitRuntimeTextInit(staged, staged+8, staged+16); err != nil {
		return err
	}
	if _, err := c.emitRuntimeDynamicTextConcat(input, staged, staged+8, slots, constants); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(3, 19, staged+8), loadIntegerSlot(5, 19, staged), loadIntegerSlot(2, 19, result.pointer), movz(6, 0)); copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(6, 3))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(7, 5, 6), loadByte(4, 7), addRegister(7, 2, 6), storeByte(4, 7), addImmediate(6, 6, 1)); c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words)); c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), addRegister(7, 2, 3), movz(4, 10), storeByte(4, 7), storeIntegerSlot(3, 19, result.length))
	c.words = append(c.words, loadIntegerSlot(0, 19, staged), loadIntegerSlot(1, 19, staged+16)); c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1) // b.ne
	return nil
}
func (c *windowCode) emitRuntimeResultTextSubstring(input value, result resultSlot, slots map[string]uint32, constants map[string]constantValue) error {
	staged := c.temporaryBufferOffset; length, err := c.emitRuntimeTextSubstring(input, staged, constants, slots, false)
	if err != nil {
		return err
	}
	c.movRegister(3, length); c.emitConsoleStackAddress(5, staged)
	c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), movz(6, 0)); copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(6, 3))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(7, 5, 6), loadByte(4, 7), addRegister(7, 2, 6), storeByte(4, 7), addImmediate(6, 6, 1)); c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words)); c.words = append(c.words, addRegister(7, 2, 3), movz(4, 10), storeByte(4, 7), storeIntegerSlot(3, 19, result.length))
	return nil
}
func (c *windowCode) emitRuntimeResultTextArrayElement(input value, result resultSlot, slots map[string]uint32, constants map[string]constantValue) error {
	if err := c.emitRuntimeTextOperand(input, slots, constants, 3, 5, c.temporaryBufferOffset+256); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), movz(6, 0)); copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(6, 3))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(7, 5, 6), loadByte(4, 7), addRegister(7, 2, 6), storeByte(4, 7), addImmediate(6, 6, 1)); c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words)); c.words = append(c.words, addRegister(7, 2, 3), movz(4, 10), storeByte(4, 7), storeIntegerSlot(3, 19, result.length))
	return nil
}
func (c *windowCode) emitRuntimeResultTextArgument(input value, result resultSlot, slots map[string]uint32, constants map[string]constantValue) error {
	if err := c.emitRuntimeArgumentTextOperand(input, slots, constants, 3, 5); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(2, 19, result.pointer), movz(6, 0)); copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(6, 3))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(7, 5, 6), loadByte(4, 7), addRegister(7, 2, 6), storeByte(4, 7), addImmediate(6, 6, 1)); c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words)); c.words = append(c.words, addRegister(7, 2, 3), movz(4, 10), storeByte(4, 7), storeIntegerSlot(3, 19, result.length))
	return nil
}
func (c *windowCode) emitRuntimeTextVariable(name string, input value, constants map[string]constantValue, slots map[string]uint32, initialize bool) error {
	buffer, length := c.textBuffers[name], c.textLengths[name]
	if input.kind == "引用" {
		if input.text == name {
			return nil
		}
		if _, exists := c.textBuffers[input.text]; !exists {
			return fmt.Errorf("无法解析文本局部复制来源：%s", input.text)
		}
	}
	if isRuntimeFileRead(input) {
		return c.emitRuntimeTextFile(name, input, constants, slots, initialize)
	}
	if isStoredTextReturnCall(input) {
		return c.emitRuntimeTextReturnStorage(name, input, constants, slots, initialize)
	}
	if initialize && !isRuntimeEnvironmentRead(input) {
		if err := c.emitRuntimeTextInit(buffer, length, c.textCapacities[name]); err != nil {
			return err
		}
	}
	if isRuntimeTextInput(input) {
		return c.emitReadDynamicText(buffer, length, c.textCapacities[name], 0, c.exitIndex)
	}
	if isRuntimeEnvironmentRead(input) {
		sourceBuffer, sourceLength, err := c.emitRuntimeEnvironmentRead(input, constants, slots)
		if err != nil {
			return err
		}
		c.words = append(c.words, storeIntegerSlot(sourceBuffer, 19, buffer), storeIntegerSlot(sourceLength, 19, length))
		return nil
	}
	if isTextResultExtraction(input) {
		return c.emitRuntimeTextResultVariableCopy(input, buffer, length, c.textCapacities[name])
	}
	if isRuntimeArgumentText(input) {
		if err := c.emitRuntimeArgumentTextOperand(input, slots, constants, 22, 21); err != nil {
			return err
		}
		return c.emitRuntimeTextCopyOperand(buffer, length, c.textCapacities[name])
	}
	if input.kind == "引用" {
		if err := c.emitRuntimeTextOperand(input, slots, constants, 22, 21, c.temporaryBufferOffset+256); err != nil {
			return err
		}
		return c.emitRuntimeTextCopyOperand(buffer, length, c.textCapacities[name])
	}
	if input.text == "textConcat" {
		_, err := c.emitRuntimeTextConcat(input, buffer, &length, slots, constants)
		return err
	}
	resultLength, err := c.emitRuntimeTextSubstring(input, buffer, constants, slots, false)
	if err == nil {
		c.words = append(c.words, storeIntegerSlot(resultLength, 19, length))
	}
	return err
}
func (c *windowCode) emitRuntimeTextReturnStorage(name string, input value, constants map[string]constantValue, slots map[string]uint32, initialize bool) error {
	buffer, length := c.textBuffers[name], c.textLengths[name]
	if initialize {
		c.words = append(c.words, movz(0, 0), storeIntegerSlot(0, 19, buffer), storeIntegerSlot(0, 19, length))
	}
	if err := c.emitRuntimeFunctionCall(input, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(1, 31, 0), storeIntegerSlot(2, 31, 8), storeIntegerSlot(3, 31, 16)); c.words = append(c.words, loadIntegerSlot(0, 19, buffer), compareImmediate(0, 0))
	noOldMapping := c.conditionalPlaceholder(1) // b.eq
	c.words = append(c.words, loadIntegerSlot(0, 19, buffer), loadIntegerSlot(1, 19, c.textCapacities[name])); c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0)); c.branchToRuntimeFailure(1)
	c.patchConditional(noOldMapping, len(c.words)); c.words = append(c.words, loadIntegerSlot(1, 31, 0), loadIntegerSlot(2, 31, 8), loadIntegerSlot(3, 31, 16), addImmediate(31, 31, 32))
	c.words = append(c.words, storeIntegerSlot(1, 19, buffer), storeIntegerSlot(2, 19, length))
	if err := c.emitRegisterImmediate(0, 16*1024*1024); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, c.textCapacities[name]))
	return nil
}
func (c *windowCode) emitConsoleStatements(statements []statement, constants map[string]constantValue, slots map[string]uint32) error {
	for _, item := range statements {
		switch item.kind {
		case "跳出":
			if len(c.loopTargets) == 0 {
				return fmt.Errorf("跳出语句没有所在循环")
			}
			position := c.unconditionalPlaceholder(); last := len(c.loopTargets) - 1
			c.loopTargets[last].breaks = append(c.loopTargets[last].breaks, position)
		case "继续":
			if len(c.loopTargets) == 0 {
				return fmt.Errorf("继续语句没有所在循环")
			}
			target := c.loopTargets[len(c.loopTargets)-1].continueAt; c.words = append(c.words, branch(target-len(c.words)))
		case "变量":
			if mapSlot, exists := c.mapSlots[item.name.text]; exists {
				c.words = append(c.words, movz(0, 0), storeIntegerSlot(0, 19, mapSlot.data), storeIntegerSlot(0, 19, mapSlot.length), movz(0, 64), storeIntegerSlot(0, 19, mapSlot.capacity))
				continue
			}
			if result, exists := c.resultSlots[item.name.text]; exists {
				if result.runtimeText {
					c.words = append(c.words, movz(0, 0), storeIntegerSlot(0, 19, result.pointer), storeIntegerSlot(0, 19, result.length))
				}
				if err := c.emitRuntimeResultValue(item.name.text, item.value, constants, slots, result); err != nil {
					return err
				}
				c.words = append(c.words, storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(1, 19, result.value))
				if result.text {
					c.words = append(c.words, storeIntegerSlot(2, 19, result.pointer), storeIntegerSlot(3, 19, result.length))
				}
				continue
			}
			if _, exists := c.textBuffers[item.name.text]; exists {
				if err := c.emitRuntimeTextVariable(item.name.text, item.value, constants, slots, true); err != nil {
					return err
				}
				continue
			}
			if array, exists := c.arraySlots[item.name.text]; exists {
				if err := c.emitRuntimeArrayInit(item.value, array, constants, slots); err != nil {
					return err
				}
				continue
			}
			if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, storeIntegerSlot(0, 19, slots[item.name.text]))
		case "赋值":
			if _, exists := c.mapSlots[item.name.text]; exists {
				if err := c.emitRuntimeMapSet(item.value, constants, slots); err != nil {
					return err
				}
				continue
			}
			if result, exists := c.resultSlots[item.name.text]; exists {
				if err := c.emitRuntimeResultValue(item.name.text, item.value, constants, slots, result); err != nil {
					return err
				}
				c.words = append(c.words, storeIntegerSlot(0, 19, result.tag), storeIntegerSlot(1, 19, result.value))
				if result.text {
					c.words = append(c.words, storeIntegerSlot(2, 19, result.pointer), storeIntegerSlot(3, 19, result.length))
				}
				continue
			}
			if item.value.kind == "调用" && (item.value.text == "arraySet" || item.value.text == "arrayPush") {
				var err error; array := c.arraySlots[item.name.text]
				if array.text && item.value.text == "arraySet" {
					err = c.emitRuntimeTextArraySet(item.value, constants, slots)
				} else if array.text {
					err = c.emitRuntimeTextArrayPush(item.value, constants, slots)
				} else if item.value.text == "arraySet" {
					err = c.emitRuntimeArraySet(item.value, constants, slots)
				} else {
					err = c.emitRuntimeArrayPush(item.value, constants, slots)
				}
				if err != nil {
					return err
				}
				continue
			}
			if _, exists := c.textBuffers[item.name.text]; exists && (isRuntimeTextValue(item.value) || isRuntimeFileRead(item.value) || isRuntimeArgumentText(item.value) || isTextResultExtraction(item.value) || isStoredTextReturnCall(item.value)) {
				if err := c.emitRuntimeTextVariable(item.name.text, item.value, constants, slots, false); err != nil {
					return err
				}
				continue
			}
			if err := c.emitRuntimeInteger(item.value, constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, storeIntegerSlot(0, 19, slots[item.name.text]))
		case "删映射键":
			if err := c.emitRuntimeMapDelete(item.value, constants, slots); err != nil {
				return err
			}
			continue
		case "若":
			staticCondition := !containsRuntimeArguments(item.condition) && !containsRuntimeIntegerInput(item.condition) && !containsRuntimeTextInput(item.condition) && !containsIntegerSlotReference(item.condition, slots) && !containsRuntimeTextVariableReference(item.condition, c.textBuffers) && !containsTextResultValue(item.condition)
			if staticCondition {
				selected, err := evaluateBoolean(item.condition, 0, constants)
				if err != nil {
					return fmt.Errorf("无法计算静态若条件：%w", err)
				}
				branchStatements := item.body
				if !selected {
					branchStatements = item.otherwise
				}
				if err := c.emitConsoleStatements(branchStatements, constants, slots); err != nil {
					return err
				}
				continue
			}
			falsePatches, err := c.emitConditionFalseBranches(item.condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitConsoleStatements(item.body, constants, slots); err != nil {
				return err
			}
			if len(item.otherwise) == 0 {
				c.patchConditionFalse(falsePatches, len(c.words))
				continue
			}
			end := c.unconditionalPlaceholder(); c.patchConditionFalse(falsePatches, len(c.words))
			if err := c.emitConsoleStatements(item.otherwise, constants, slots); err != nil {
				return err
			}
			c.words[end] = branch(len(c.words) - end)
		case "凡":
			staticCondition := !containsRuntimeArguments(item.condition) && !containsRuntimeIntegerInput(item.condition) && !containsRuntimeTextInput(item.condition) && !containsIntegerSlotReference(item.condition, slots) && !containsRuntimeTextVariableReference(item.condition, c.textBuffers) && !containsTextResultValue(item.condition)
			if staticCondition {
				selected, err := evaluateBoolean(item.condition, 0, constants)
				if err != nil {
					return fmt.Errorf("无法计算静态凡条件：%w", err)
				}
				if !selected {
					continue
				}
				loopStart := len(c.words); c.loopTargets = append(c.loopTargets, loopTarget{continueAt: loopStart})
				if err := c.emitConsoleStatements(item.body, constants, slots); err != nil {
					return err
				}
				c.words = append(c.words, branch(loopStart-len(c.words))); c.finishLoop()
				continue
			}
			loopStart := len(c.words); c.loopTargets = append(c.loopTargets, loopTarget{continueAt: loopStart})
			falsePatches, err := c.emitConditionFalseBranches(item.condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitConsoleStatements(item.body, constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, branch(loopStart-len(c.words))); c.patchConditionFalse(falsePatches, len(c.words))
			c.finishLoop()
		default:
			return fmt.Errorf("不支持的控制台运行时语句：%s", item.kind)
		}
	}
	return nil
}
func (c *windowCode) emitRuntimeTextConcat(input value, destinationBuffer uint32, destinationLength *uint32, slots map[string]uint32, constants map[string]constantValue) (uint32, error) {
	if input.kind != "调用" || input.text != "textConcat" || len(input.args) != 2 {
		return 0, fmt.Errorf("有界文本拼接需要 textConcat（文本，文本）")
	}
	if destinationLength != nil && c.isRuntimeTextPointerSlot(destinationBuffer) {
		return c.emitRuntimeDynamicTextConcat(input, destinationBuffer, *destinationLength, slots, constants)
	}
	scratchBase := c.temporaryBufferOffset + runtimeTextConcatBaseScratch + runtimeTextConcatScratchStride*(runtimeTextConcatScratchLevel(input)-1); leftScratch, rightScratch := scratchBase, scratchBase+256
	if err := c.emitRuntimeTextOperand(input.args[0], slots, constants, 21, 24, leftScratch); err != nil {
		return 0, err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(21, 31, 0), storeIntegerSlot(24, 31, 8))
	if err := c.emitRuntimeTextOperand(input.args[1], slots, constants, 22, 27, rightScratch); err != nil {
		return 0, err
	}
	c.words = append(c.words, loadIntegerSlot(21, 31, 0), loadIntegerSlot(24, 31, 8), addImmediate(31, 31, 16)); c.words = append(c.words, addRegister(23, 21, 22), compareImmediate(23, 63))
	c.branchToRuntimeFailure(12) // b.gt: the result exceeds 63 input bytes
	c.words = append(c.words, movz(25, 0)); c.emitConsoleStackAddress(25, c.temporaryBufferOffset)
	c.words = append(c.words, movz(26, 0)); leftLoop := len(c.words)
	c.words = append(c.words, compareRegister(26, 21))
	leftDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(1, 24, 26), loadByte(28, 1), addRegister(1, 25, 26), storeByte(28, 1), addImmediate(26, 26, 1)); c.words = append(c.words, branch(leftLoop-len(c.words)))
	c.patchConditional(leftDone, len(c.words)); c.words = append(c.words, movz(26, 0))
	rightLoop := len(c.words); c.words = append(c.words, compareRegister(26, 22))
	rightDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(1, 27, 26), loadByte(28, 1), addRegister(1, 25, 21), addRegister(1, 1, 26), storeByte(28, 1), addImmediate(26, 26, 1)); c.words = append(c.words, branch(rightLoop-len(c.words)))
	c.patchConditional(rightDone, len(c.words))
	c.words = append(c.words, addRegister(1, 25, 23))
	c.movImmediate(28, 10)
	c.words = append(c.words, storeByte(28, 1))
	c.emitConsoleStackAddress(24, c.temporaryBufferOffset)
	if c.isRuntimeTextPointerSlot(destinationBuffer) {
		c.words = append(c.words, loadIntegerSlot(25, 19, destinationBuffer))
	} else {
		c.emitConsoleStackAddress(25, destinationBuffer)
	}
	c.words = append(c.words, movz(26, 0))
	copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(26, 23))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(27, 24, 26), loadByte(28, 27), addRegister(27, 25, 26), storeByte(28, 27), addImmediate(26, 26, 1))
	c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words))
	c.words = append(c.words, addRegister(27, 25, 23))
	c.movImmediate(28, 10)
	c.words = append(c.words, storeByte(28, 27))
	if destinationLength != nil {
		c.words = append(c.words, storeIntegerSlot(23, 19, *destinationLength))
	}
	return 23, nil
}
func (c *windowCode) emitRuntimeDynamicTextConcat(input value, destinationPointer, destinationLength uint32, slots map[string]uint32, constants map[string]constantValue) (uint32, error) {
	const maximum = 16 * 1024 * 1024
	scratchBase := c.temporaryBufferOffset + runtimeTextConcatBaseScratch + runtimeTextConcatScratchStride*(runtimeTextConcatScratchLevel(input)-1)
	leftScratch, rightScratch := scratchBase, scratchBase+256
	if err := c.emitRuntimeTextOperand(input.args[0], slots, constants, 21, 24, leftScratch); err != nil {
		return 0, err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(21, 31, 0), storeIntegerSlot(24, 31, 8))
	if err := c.emitRuntimeTextOperand(input.args[1], slots, constants, 22, 27, rightScratch); err != nil {
		return 0, err
	}
	c.words = append(c.words, loadIntegerSlot(21, 31, 0), loadIntegerSlot(24, 31, 8), addImmediate(31, 31, 16))
	c.words = append(c.words, addRegister(23, 21, 22))
	if err := c.emitRegisterImmediate(4, maximum-1); err != nil {
		return 0, err
	}
	c.words = append(c.words, compareRegister(23, 4))
	c.branchToRuntimeFailure(12) // b.gt: reserve one byte for the line terminator
	capacitySlot := destinationPointer + 16
	c.words = append(c.words, addImmediate(4, 23, 1), loadIntegerSlot(26, 19, capacitySlot))
	c.words = append(c.words, compareRegister(26, 4))
	capacityReady := c.conditionalPlaceholder(2) // b.hs
	capacityLoop := len(c.words)
	c.words = append(c.words, compareRegister(26, 4))
	capacityEnough := c.conditionalPlaceholder(2) // b.hs
	if err := c.emitRegisterImmediate(5, maximum/2); err != nil {
		return 0, err
	}
	c.words = append(c.words, compareRegister(26, 5))
	capacityAtLimit := c.conditionalPlaceholder(12) // b.gt
	c.words = append(c.words, logicalShiftLeft(26, 26, 1))
	c.words = append(c.words, branch(capacityLoop-len(c.words)))
	setMaximum := len(c.words)
	if err := c.emitRegisterImmediate(26, maximum); err != nil {
		return 0, err
	}
	capacityReadyTarget := len(c.words)
	c.patchConditional(capacityReady, capacityReadyTarget)
	c.patchConditional(capacityEnough, capacityReadyTarget)
	c.patchConditional(capacityAtLimit, setMaximum)
	if err := c.emitArrayMapCall(26); err != nil {
		return 0, err
	}
	c.movRegister(28, 0)
	c.words = append(c.words, movz(7, 0))
	leftLoop := len(c.words)
	c.words = append(c.words, compareRegister(7, 21))
	leftDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(1, 24, 7), loadByte(0, 1), addRegister(1, 28, 7), storeByte(0, 1), addImmediate(7, 7, 1))
	c.words = append(c.words, branch(leftLoop-len(c.words)))
	c.patchConditional(leftDone, len(c.words))
	c.words = append(c.words, movz(7, 0))
	rightLoop := len(c.words)
	c.words = append(c.words, compareRegister(7, 22))
	rightDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(1, 27, 7), loadByte(0, 1), addRegister(1, 28, 21), addRegister(1, 1, 7), storeByte(0, 1), addImmediate(7, 7, 1))
	c.words = append(c.words, branch(rightLoop-len(c.words)))
	c.patchConditional(rightDone, len(c.words))
	c.words = append(c.words, addRegister(1, 28, 23))
	c.movImmediate(0, 10)
	c.words = append(c.words, storeByte(0, 1))
	c.words = append(c.words, loadIntegerSlot(0, 19, destinationPointer))
	c.movRegister(1, 26)
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1) // b.ne
	c.words = append(c.words, storeIntegerSlot(28, 19, destinationPointer), storeIntegerSlot(26, 19, capacitySlot), storeIntegerSlot(23, 19, destinationLength))
	return 23, nil
}
func (c *windowCode) emitRuntimeArgumentTextOperand(input value, slots map[string]uint32, constants map[string]constantValue, lengthRegister, bufferRegister uint32) error {
	if !isRuntimeArgumentText(input) {
		return fmt.Errorf("命令行文本需要 argumentText（整数下标）")
	}
	if err := c.emitRuntimeInteger(input.args[0], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(0, 31, 0), compareImmediate(0, 0))
	negative := c.conditionalPlaceholder(11)
	c.words = append(c.words, loadIntegerSlot(1, 19, c.argumentCountOffset), compareRegister(0, 1))
	absent := c.conditionalPlaceholder(10)
	c.words = append(c.words, loadIntegerSlot(1, 19, c.argumentVectorOffset), loadIndexedRegister(bufferRegister, 1, 0), addImmediate(31, 31, 16), movz(lengthRegister, 0))
	scan := len(c.words)
	if err := c.emitRegisterImmediate(0, 4096); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(lengthRegister, 0))
	overflow := c.conditionalPlaceholder(10)
	c.words = append(c.words, addRegister(0, bufferRegister, lengthRegister), loadByte(0, 0), compareImmediate(0, 0))
	done := c.conditionalPlaceholder(0)
	c.words = append(c.words, addImmediate(lengthRegister, lengthRegister, 1))
	c.words = append(c.words, branch(scan-len(c.words)))
	c.patchConditional(done, len(c.words))
	exitJump := c.unconditionalPlaceholder()
	c.patchConditional(overflow, len(c.words))
	c.words = append(c.words, addImmediate(31, 31, 16))
	c.movImmediate(0, 3)
	c.call(c.exitIndex)
	c.patchConditional(negative, len(c.words))
	c.patchConditional(absent, len(c.words))
	c.words = append(c.words, addImmediate(31, 31, 16), movz(0, 2))
	c.call(c.exitIndex)
	c.words[exitJump] = branch(len(c.words) - exitJump)
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(bufferRegister, 31, 0), storeIntegerSlot(lengthRegister, 31, 8))
	c.movRegister(0, bufferRegister)
	c.movRegister(1, lengthRegister)
	c.emitRuntimeUTF8Validation(0, 1)
	c.words = append(c.words, loadIntegerSlot(bufferRegister, 31, 0), loadIntegerSlot(lengthRegister, 31, 8), addImmediate(31, 31, 16))
	return nil
}
func (c *windowCode) emitRuntimeTextCopyOperand(buffer, length, capacity uint32) error {
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(21, 31, 0), storeIntegerSlot(22, 31, 8), addImmediate(23, 22, 1), loadIntegerSlot(24, 19, capacity), compareRegister(24, 23))
	ready := c.conditionalPlaceholder(2)
	if err := c.emitArrayMapCall(23); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 16), loadIntegerSlot(0, 19, buffer), loadIntegerSlot(1, 19, capacity))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, loadIntegerSlot(0, 31, 16), storeIntegerSlot(0, 19, buffer), storeIntegerSlot(23, 19, capacity))
	c.patchConditional(ready, len(c.words))
	c.words = append(c.words, loadIntegerSlot(21, 31, 0), loadIntegerSlot(22, 31, 8), loadIntegerSlot(24, 19, buffer), movz(23, 0))
	loop := len(c.words)
	c.words = append(c.words, compareRegister(23, 22))
	done := c.conditionalPlaceholder(10)
	c.words = append(c.words, addRegister(25, 21, 23), loadByte(26, 25), addRegister(25, 24, 23), storeByte(26, 25), addImmediate(23, 23, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(done, len(c.words))
	c.words = append(c.words, addRegister(25, 24, 22))
	c.movImmediate(26, 10)
	c.words = append(c.words, storeByte(26, 25), storeIntegerSlot(22, 19, length), addImmediate(31, 31, 32))
	return nil
}
func (c *windowCode) emitRuntimeTextCopyRegisters(buffer, length, capacity, sourceLength, sourceBuffer uint32) error {
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(sourceLength, 31, 0), storeIntegerSlot(sourceBuffer, 31, 8), addImmediate(23, sourceLength, 1), loadIntegerSlot(24, 19, capacity), compareRegister(24, 23))
	ready := c.conditionalPlaceholder(2)
	if err := c.emitArrayMapCall(23); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 16), loadIntegerSlot(0, 19, buffer), loadIntegerSlot(1, 19, capacity))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, loadIntegerSlot(0, 31, 16), storeIntegerSlot(0, 19, buffer), storeIntegerSlot(23, 19, capacity))
	c.patchConditional(ready, len(c.words))
	c.words = append(c.words, loadIntegerSlot(21, 31, 0), loadIntegerSlot(22, 31, 8), loadIntegerSlot(24, 19, buffer), movz(23, 0))
	loop := len(c.words)
	c.words = append(c.words, compareRegister(23, 22))
	done := c.conditionalPlaceholder(10)
	c.words = append(c.words, addRegister(25, 21, 23), loadByte(26, 25), addRegister(25, 24, 23), storeByte(26, 25), addImmediate(23, 23, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(done, len(c.words))
	c.words = append(c.words, addRegister(25, 24, 22))
	c.movImmediate(26, 10)
	c.words = append(c.words, storeByte(26, 25), storeIntegerSlot(22, 19, length), addImmediate(31, 31, 32))
	return nil
}
func (c *windowCode) emitRuntimeOwnedTextReturnOperand(input value, constants map[string]constantValue, slots map[string]uint32, lengthRegister, bufferRegister uint32) error {
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(1, 31, 0), storeIntegerSlot(2, 31, 8), storeIntegerSlot(3, 31, 16))
	if err := c.emitRuntimeFunctionCall(input, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(1, 31, 0), storeIntegerSlot(2, 31, 8), storeIntegerSlot(3, 31, 0))
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), loadIntegerSlot(1, 31, 8), compareRegister(0, 1))
	failure := c.conditionalPlaceholder(1)
	c.words = append(c.words, loadIntegerSlot(lengthRegister, 31, 0), loadIntegerSlot(bufferRegister, 31, 8))
	c.words = append(c.words, compareRegister(lengthRegister, 0))
	noMapping := c.conditionalPlaceholder(1)
	c.movRegister(0, bufferRegister)
	c.movImmediate(1, 0)
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.patchConditional(failure, len(c.words))
	c.patchConditional(noMapping, len(c.words))
	c.words = append(c.words, loadIntegerSlot(1, 31, 0), loadIntegerSlot(0, 31, 8), addImmediate(31, 31, 16))
	return nil
}
func (c *windowCode) emitRuntimeTextOperand(input value, slots map[string]uint32, constants map[string]constantValue, lengthRegister, bufferRegister, scratchBuffer uint32) error {
	if input.kind == "引用" {
		if constant, exists := constants[input.text]; exists && constant.kind == "字符串" {
			text := constant.text
			label := fmt.Sprintf("runtimeTextConcat%d", len(c.dataLabels))
			c.addData(label, []byte(text), 1)
			c.movImmediate(lengthRegister, uint16(len(text)))
			c.address(label, bufferRegister)
			return nil
		}
		buffer, bufferExists := c.textBuffers[input.text]
		length, lengthExists := c.textLengths[input.text]
		if !bufferExists || !lengthExists {
			return fmt.Errorf("无法解析运行时拼接文本变量：%s", input.text)
		}
		c.words = append(c.words, loadIntegerSlot(lengthRegister, 19, length), loadIntegerSlot(bufferRegister, 19, buffer))
		return nil
	}
	if input.kind == "调用" && input.text == "结果值" && len(input.args) == 1 && input.args[0].kind == "引用" {
		result, exists := c.resultSlots[input.args[0].text]
		if !exists || !result.text {
			return fmt.Errorf("无法解析文本结果变量：%s", input.args[0].text)
		}
		c.words = append(c.words, loadIntegerSlot(2, 19, result.tag), compareImmediate(2, 1))
		success := c.conditionalPlaceholder(1)
		c.words = append(c.words, loadIntegerSlot(0, 19, result.value))
		c.call(c.exitIndex)
		c.patchConditional(success, len(c.words))
		c.words = append(c.words, loadIntegerSlot(lengthRegister, 19, result.length), loadIntegerSlot(bufferRegister, 19, result.pointer))
		return nil
	}
	if isDirectTextResultExtraction(input) {
		if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, compareImmediate(0, 1))
		success := c.conditionalPlaceholder(1)
		c.movRegister(0, 1)
		c.call(c.exitIndex)
		c.patchConditional(success, len(c.words))
		c.movRegister(lengthRegister, 3)
		c.movRegister(bufferRegister, 2)
		return nil
	}
	if isRuntimeTextInput(input) {
		if err := c.emitReadText(scratchBuffer, nil, true, 0, c.exitIndex-1, c.exitIndex); err != nil {
			return err
		}
		c.movRegister(lengthRegister, 22)
		c.emitConsoleStackAddress(bufferRegister, scratchBuffer)
		return nil
	}
	if isRuntimeArgumentText(input) {
		return c.emitRuntimeArgumentTextOperand(input, slots, constants, lengthRegister, bufferRegister)
	}
	if isRuntimeEnvironmentRead(input) {
		sourceBuffer, sourceLength, err := c.emitRuntimeEnvironmentRead(input, constants, slots)
		if err != nil {
			return err
		}
		c.movRegister(lengthRegister, sourceLength)
		c.movRegister(bufferRegister, sourceBuffer)
		return nil
	}
	if input.kind == "调用" && input.text == "substring" {
		length, err := c.emitRuntimeTextSubstring(input, scratchBuffer, constants, slots, false)
		if err != nil {
			return err
		}
		c.movRegister(lengthRegister, length)
		c.emitConsoleStackAddress(bufferRegister, scratchBuffer)
		return nil
	}
	if input.kind == "调用" && input.text == "arrayGet" && len(input.args) == 2 && input.args[0].kind == "引用" {
		array, exists := c.arraySlots[input.args[0].text]
		if !exists || !array.text {
			return fmt.Errorf("运行时文本索引需要字符串数组变量")
		}
		if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
			return err
		}
		c.emitRuntimeArrayIndexBounds(array.length)
		c.words = append(c.words, logicalShiftLeft(2, 0, 1), loadIntegerSlot(1, 19, array.data), loadIndexedRegister(bufferRegister, 1, 2), addImmediate(1, 1, 8), loadIndexedRegister(lengthRegister, 1, 2))
		return nil
	}
	if input.kind == "调用" && input.text == "textConcat" {
		lengthOffset := scratchBuffer + 64
		length, err := c.emitRuntimeTextConcat(input, scratchBuffer, &lengthOffset, slots, constants)
		if err != nil {
			return err
		}
		c.movRegister(lengthRegister, length)
		c.emitConsoleStackAddress(bufferRegister, scratchBuffer)
		return nil
	}
	if isTextReturnCall(input) {
		return c.emitRuntimeOwnedTextReturnOperand(input, constants, slots, lengthRegister, bufferRegister)
	}
	text, problem := evaluateString(input, 0, constants)
	if problem != nil {
		return fmt.Errorf("无法计算运行时拼接静态文本：%w", problem)
	}
	label := fmt.Sprintf("runtimeTextConcat%d", len(c.dataLabels))
	c.addData(label, []byte(text), 1)
	c.movImmediate(lengthRegister, uint16(len(text)))
	c.address(label, bufferRegister)
	return nil
}
func (c *windowCode) emitRuntimeTextSubstring(input value, destinationBuffer uint32, constants map[string]constantValue, slots map[string]uint32, boundedOutput bool) (uint32, error) {
	if input.kind != "调用" || input.text != "substring" || len(input.args) != 3 {
		return 0, fmt.Errorf("有界运行时切片需要 substring（文本，整数起点，整数终点）")
	}
	source := input.args[0]
	if source.kind != "引用" && !isRuntimeArgumentText(source) && !(source.kind == "调用" && source.text == "结果值" && len(source.args) == 1 && source.args[0].kind == "引用") && (source.kind != "调用" || source.text != "textConcat" && source.text != "arrayGet" && source.text != "readText") {
		return 0, fmt.Errorf("有界运行时切片源必须是文本变量、参数文本、文本结果值或 textConcat 结果")
	}
	if err := c.emitRuntimeTextOperand(source, slots, constants, 21, 17, c.temporaryBufferOffset+256); err != nil {
		return 0, fmt.Errorf("无法生成运行时切片源：%w", err)
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(21, 31, 0), storeIntegerSlot(17, 31, 8))
	for index, register := range []uint32{6, 7} {
		if err := c.emitRuntimeInteger(input.args[index+1], constants, slots); err != nil {
			return 0, fmt.Errorf("无法生成运行时切片索引：%w", err)
		}
		c.movRegister(register, 0)
	}
	c.words = append(c.words, loadIntegerSlot(21, 31, 0), loadIntegerSlot(17, 31, 8), addImmediate(31, 31, 16))
	c.words = append(c.words, compareImmediate(6, 0))
	c.branchToRuntimeFailure(11) // b.lt: negative start index
	c.words = append(c.words, compareRegister(7, 6))
	c.branchToRuntimeFailure(11) // b.lt: end precedes start
	c.words = append(c.words, compareImmediate(7, 63))
	c.branchToRuntimeFailure(12) // b.gt: byte-bounded text has at most 63 scalars
	c.words = append(c.words, movz(22, 0), movz(23, 0), movz(24, 0), movz(26, 0))
	scan := len(c.words)
	c.words = append(c.words, compareRegister(23, 6))
	skipStart := c.conditionalPlaceholder(1) // b.ne
	c.movRegister(24, 22)
	c.patchConditional(skipStart, len(c.words))
	c.words = append(c.words, compareRegister(23, 7))
	skipEnd := c.conditionalPlaceholder(1) // b.ne
	c.movRegister(26, 22)
	c.patchConditional(skipEnd, len(c.words))
	c.words = append(c.words, compareRegister(23, 7))
	finished := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, compareRegister(22, 21))
	c.branchToRuntimeFailure(2) // b.hs: the input ended before the requested codepoint
	c.words = append(c.words, addRegister(27, 17, 22), loadByte(28, 27), compareImmediate(28, 0x80))
	ascii := c.conditionalPlaceholder(3) // b.lo
	c.words = append(c.words, compareImmediate(28, 0xe0))
	twoBytes := c.conditionalPlaceholder(3) // b.lo
	c.words = append(c.words, compareImmediate(28, 0xf0))
	threeBytes := c.conditionalPlaceholder(3) // b.lo
	c.words = append(c.words, addImmediate(22, 22, 4))
	advanceBranches := []int{c.unconditionalPlaceholder()}
	asciiIndex := len(c.words)
	c.patchConditional(ascii, asciiIndex)
	c.words = append(c.words, addImmediate(22, 22, 1))
	advanceBranches = append(advanceBranches, c.unconditionalPlaceholder())
	twoBytesIndex := len(c.words)
	c.patchConditional(twoBytes, twoBytesIndex)
	c.words = append(c.words, addImmediate(22, 22, 2))
	advanceBranches = append(advanceBranches, c.unconditionalPlaceholder())
	threeBytesIndex := len(c.words)
	c.patchConditional(threeBytes, threeBytesIndex)
	c.words = append(c.words, addImmediate(22, 22, 3))
	advanceBranches = append(advanceBranches, c.unconditionalPlaceholder())
	advance := len(c.words)
	for _, position := range advanceBranches {
		c.words[position] = branch(advance - position)
	}
	c.words = append(c.words, addImmediate(23, 23, 1))
	scanBranch := len(c.words)
	c.words = append(c.words, branch(scan-scanBranch))
	scanDone := len(c.words)
	c.patchConditional(finished, scanDone)
	c.words = append(c.words, subtractRegister(23, 26, 24))
	c.movRegister(22, 17)
	c.words = append(c.words, addRegister(22, 22, 24))
	if c.isRuntimeTextPointerSlot(destinationBuffer) {
		if err := c.emitRuntimeDynamicTextSubstring(destinationBuffer); err != nil {
			return 0, err
		}
		return 23, nil
	}
	if boundedOutput {
		c.words = append(c.words, compareImmediate(23, 63))
		c.branchToRuntimeFailure(12) // b.gt: root output scratch is limited to 63 bytes
	}
	c.emitConsoleStackAddress(24, destinationBuffer)
	c.words = append(c.words, movz(25, 0))
	copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(25, 23))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(27, 22, 25), loadByte(28, 27), addRegister(27, 24, 25), storeByte(28, 27), addImmediate(25, 25, 1))
	copyBranch := len(c.words)
	c.words = append(c.words, branch(copyLoop-copyBranch))
	c.patchConditional(copyDone, len(c.words))
	c.words = append(c.words, addRegister(27, 24, 23))
	c.movImmediate(28, 10)
	c.words = append(c.words, storeByte(28, 27))
	return 23, nil
}
func (c *windowCode) emitRuntimeDynamicTextSubstring(destinationPointer uint32) error {
	const maximum = 16 * 1024 * 1024
	capacitySlot := destinationPointer + 16
	c.words = append(c.words, addImmediate(4, 23, 1), loadIntegerSlot(26, 19, capacitySlot), compareRegister(26, 4))
	capacityReady := c.conditionalPlaceholder(2) // b.hs
	capacityLoop := len(c.words)
	c.words = append(c.words, compareRegister(26, 4))
	capacityEnough := c.conditionalPlaceholder(2) // b.hs
	if err := c.emitRegisterImmediate(5, maximum/2); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(26, 5))
	capacityAtLimit := c.conditionalPlaceholder(12) // b.gt
	c.words = append(c.words, logicalShiftLeft(26, 26, 1))
	c.words = append(c.words, branch(capacityLoop-len(c.words)))
	setMaximum := len(c.words)
	if err := c.emitRegisterImmediate(26, maximum); err != nil {
		return err
	}
	capacityReadyTarget := len(c.words)
	c.patchConditional(capacityReady, capacityReadyTarget)
	c.patchConditional(capacityEnough, capacityReadyTarget)
	c.patchConditional(capacityAtLimit, setMaximum)
	if err := c.emitArrayMapCall(26); err != nil {
		return err
	}
	c.movRegister(28, 0)
	c.words = append(c.words, movz(7, 0))
	copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(7, 23))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(1, 22, 7), loadByte(0, 1), addRegister(1, 28, 7), storeByte(0, 1), addImmediate(7, 7, 1))
	c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words))
	c.words = append(c.words, addRegister(1, 28, 23))
	c.movImmediate(0, 10)
	c.words = append(c.words, storeByte(0, 1), loadIntegerSlot(0, 19, destinationPointer), loadIntegerSlot(1, 19, capacitySlot))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1) // b.ne
	c.words = append(c.words, storeIntegerSlot(28, 19, destinationPointer), storeIntegerSlot(26, 19, capacitySlot))
	return nil
}
func (c *windowCode) isRuntimeTextPointerSlot(offset uint32) bool {
	for _, slot := range c.textBuffers {
		if slot == offset {
			return true
		}
	}
	return false
}
func runtimeTextSubstringNeedsScratch(input value) bool {
	if input.kind == "调用" && input.text == "substring" {
		return true
	}
	for _, argument := range input.args {
		if runtimeTextSubstringNeedsScratch(argument) {
			return true
		}
	}
	if input.left != nil && runtimeTextSubstringNeedsScratch(*input.left) || input.right != nil && runtimeTextSubstringNeedsScratch(*input.right) {
		return true
	}
	for _, item := range input.fields {
		if runtimeTextSubstringNeedsScratch(item.value) {
			return true
		}
	}
	return false
}
func statementsNeedRuntimeTextSubstringScratch(items []statement) bool {
	for _, item := range items {
		if runtimeTextSubstringNeedsScratch(item.value) || runtimeTextSubstringNeedsScratch(item.condition) || statementsNeedRuntimeTextSubstringScratch(item.body) || statementsNeedRuntimeTextSubstringScratch(item.otherwise) {
			return true
		}
	}
	return false
}
func runtimeTextConcatScratchLevel(input value) uint32 {
	level := uint32(0)
	pending := []struct {
		expression value
		depth      uint32
	}{{expression: input}}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current.expression.kind == "调用" && current.expression.text == "textConcat" {
			current.depth++
			if current.depth > level {
				level = current.depth
			}
		}
		for _, argument := range current.expression.args {
			pending = append(pending, struct {
				expression value
				depth      uint32
			}{argument, current.depth})
		}
		if current.expression.left != nil {
			pending = append(pending, struct {
				expression value
				depth      uint32
			}{*current.expression.left, current.depth})
		}
		if current.expression.right != nil {
			pending = append(pending, struct {
				expression value
				depth      uint32
			}{*current.expression.right, current.depth})
		}
		for _, item := range current.expression.fields {
			pending = append(pending, struct {
				expression value
				depth      uint32
			}{item.value, current.depth})
		}
	}
	return level
}
func statementsNeedRuntimeTextConcatScratch(items []statement) uint32 {
	level := uint32(0)
	for len(items) > 0 {
		item := items[0]
		items = items[1:]
		for _, expression := range []value{item.value, item.condition} {
			if nested := runtimeTextConcatScratchLevel(expression); nested > level {
				level = nested
			}
		}
		items = append(append(items, item.body...), item.otherwise...)
	}
	return level
}
func (c *windowCode) isRuntimeTextComparisonOperand(input value, constants map[string]constantValue, slots map[string]uint32) bool {
	if isTextReturnCall(input) {
		return true
	}
	if input.kind == "调用" && input.text == "结果值" && len(input.args) == 1 && input.args[0].kind == "引用" {
		result, exists := c.resultSlots[input.args[0].text]
		return exists && result.text
	}
	kind, problem := constantType(input, constants)
	if problem == nil {
		return kind == "运行时字符串"
	}
	if input.kind == "引用" {
		_, exists := c.textBuffers[input.text]
		return exists
	}
	if input.kind != "调用" {
		return false
	}
	switch input.text {
	case "readText":
		return len(input.args) == 0
	case "argumentText":
		return len(input.args) == 1
	case "textConcat":
		for _, argument := range input.args {
			if c.isRuntimeTextComparisonOperand(argument, constants, slots) {
				return true
			}
		}
	case "substring":
		if len(input.args) == 3 && c.isRuntimeTextComparisonOperand(input.args[0], constants, slots) {
			return true
		}
		for _, argument := range input.args[1:] {
			_, runtimeSlot := slots[argument.text]
			if argument.kind == "调用" && argument.text == "readInteger" || argument.kind == "引用" && runtimeSlot {
				return true
			}
		}
	case "arrayGet":
		if len(input.args) == 2 && input.args[0].kind == "引用" {
			if array, exists := c.arraySlots[input.args[0].text]; exists && array.text {
				return true
			}
		}
	}
	return false
}
func (c *windowCode) runtimeTextComparisonNeedsScratch(input value, constants map[string]constantValue, slots map[string]uint32) bool {
	if input.kind == "调用" && (input.text == "equals" || input.text == "notEquals") && len(input.args) == 2 {
		if c.isRuntimeTextComparisonOperand(input.args[0], constants, slots) || c.isRuntimeTextComparisonOperand(input.args[1], constants, slots) {
			return true
		}
	}
	if input.kind == "比较" && isTextOrderingOperator(input.operator) && input.left != nil && input.right != nil &&
		(c.isRuntimeTextComparisonOperand(*input.left, constants, slots) || c.isRuntimeTextComparisonOperand(*input.right, constants, slots)) {
		return true
	}
	for _, child := range input.args {
		if c.runtimeTextComparisonNeedsScratch(child, constants, slots) {
			return true
		}
	}
	if input.left != nil && c.runtimeTextComparisonNeedsScratch(*input.left, constants, slots) ||
		input.right != nil && c.runtimeTextComparisonNeedsScratch(*input.right, constants, slots) {
		return true
	}
	for _, field := range input.fields {
		if c.runtimeTextComparisonNeedsScratch(field.value, constants, slots) {
			return true
		}
	}
	return false
}
func (c *windowCode) statementsNeedRuntimeTextComparisonScratch(items []statement, constants map[string]constantValue, slots map[string]uint32) bool {
	for _, item := range items {
		if c.runtimeTextComparisonNeedsScratch(item.value, constants, slots) || c.runtimeTextComparisonNeedsScratch(item.condition, constants, slots) ||
			c.statementsNeedRuntimeTextComparisonScratch(item.body, constants, slots) || c.statementsNeedRuntimeTextComparisonScratch(item.otherwise, constants, slots) {
			return true
		}
	}
	return false
}
func runtimeResultOperandShape(input value) bool {
	if input.kind == "引用" {
		return true
	}
	if input.kind != "调用" {
		return false
	}
	if input.runtimeFunction && len(input.runtimeArgumentTypes) == 0 {
		return true
	}
	if input.runtimeFunctionResult || input.text == "成功" || input.text == "失败" || isModuleReadResult(input) {
		return true
	}
	return input.text == "ifElse" && len(input.args) == 3 && (runtimeResultOperandShape(input.args[1]) || runtimeResultOperandShape(input.args[2]))
}
func (c *windowCode) emitConditionFalseBranches(condition value, constants map[string]constantValue, slots map[string]uint32) ([]conditionFalsePatch, error) {
	if !containsRuntimeArguments(condition) && !containsRuntimeIntegerInput(condition) && !containsRuntimeTextInput(condition) && !containsIntegerSlotReference(condition, slots) && !containsRuntimeTextVariableReference(condition, c.textBuffers) && !containsTextResultValue(condition) && !containsRuntimeTextReturnCall(condition) && !containsResultStateCall(condition) {
		selected, err := evaluateBoolean(condition, 0, constants)
		if err != nil {
			return nil, fmt.Errorf("无法计算静态布尔条件：%w", err)
		}
		if selected {
			return nil, nil
		}
		position := c.unconditionalPlaceholder()
		return []conditionFalsePatch{{position: position, unconditional: true}}, nil
	}
	if condition.kind == "调用" && condition.text == "映射有" && len(condition.args) == 2 && condition.args[0].kind == "引用" {
		if err := c.emitRuntimeMapHas(condition, constants, slots); err != nil {
			return nil, err
		}
		c.words = append(c.words, compareImmediate(0, 0))
		patch := c.conditionalPlaceholder(0)
		return []conditionFalsePatch{{position: patch.position, condition: patch.condition}}, nil
	}
	if condition.kind == "调用" && condition.text == "是否失败" && len(condition.args) == 1 && isResultOperand(condition.args[0]) {
		if condition.args[0].kind == "引用" {
			result, exists := c.resultSlots[condition.args[0].text]
			if !exists {
				return nil, fmt.Errorf("无法解析结果变量：%s", condition.args[0].text)
			}
			c.words = append(c.words, loadIntegerSlot(0, 19, result.tag), compareImmediate(0, 1))
			patch := c.conditionalPlaceholder(1)
			return []conditionFalsePatch{{position: patch.position, condition: patch.condition}}, nil
		} else {
			if err := c.emitResultValueRegisters(condition.args[0], constants, slots); err != nil {
				return nil, err
			}
			c.emitResultFailureFlag(runtimeResultPayload(condition.args[0], constants) == "运行时文本")
			c.words = append(c.words, compareImmediate(0, 0))
			patch := c.conditionalPlaceholder(0)
			return []conditionFalsePatch{{position: patch.position, condition: patch.condition}}, nil
		}
	}
	if condition.kind == "逻辑" && condition.left != nil && condition.right != nil {
		left, err := c.emitConditionFalseBranches(*condition.left, constants, slots)
		if err != nil {
			return nil, err
		}
		if condition.operator == "且" {
			right, issue := c.emitConditionFalseBranches(*condition.right, constants, slots)
			if issue != nil {
				return nil, issue
			}
			return append(left, right...), nil
		}
		if condition.operator != "故" {
			return nil, fmt.Errorf("不支持的运行时逻辑运算符：%s", condition.operator)
		}
		leftTrue := c.unconditionalPlaceholder()
		rightStart := len(c.words)
		c.patchConditionFalse(left, rightStart)
		right, issue := c.emitConditionFalseBranches(*condition.right, constants, slots)
		if issue != nil {
			return nil, issue
		}
		c.words[leftTrue] = branch(len(c.words) - leftTrue)
		return right, nil
	}
	if condition.kind == "调用" && (condition.text == "equals" || condition.text == "notEquals") && len(condition.args) == 2 {
		if c.isRuntimeTextComparisonOperand(condition.args[0], constants, slots) || c.isRuntimeTextComparisonOperand(condition.args[1], constants, slots) {
			return c.emitRuntimeTextComparisonFalseBranches(condition, slots, constants)
		}
	}
	if isTextReturnCall(condition) {
		if err := c.emitRuntimeInteger(condition, constants, slots); err != nil {
			return nil, err
		}
		c.words = append(c.words, compareImmediate(0, 0))
		patch := c.conditionalPlaceholder(0) // b.eq when value is false/zero
		return []conditionFalsePatch{{position: patch.position, condition: patch.condition}}, nil
	}
	if condition.kind == "比较" && condition.left != nil && condition.right != nil && isTextOrderingOperator(condition.operator) {
		if c.isRuntimeTextComparisonOperand(*condition.left, constants, slots) || c.isRuntimeTextComparisonOperand(*condition.right, constants, slots) {
			return c.emitRuntimeTextOrderingFalseBranches(condition, slots, constants)
		}
	}
	branchWhenFalse, err := c.emitRuntimeIntegerComparison(condition, constants, slots)
	if err != nil {
		return nil, err
	}
	patch := c.conditionalPlaceholder(branchWhenFalse)
	return []conditionFalsePatch{{position: patch.position, condition: patch.condition}}, nil
}
func isTextOrderingOperator(operator string) bool { return operator == "<" || operator == "<=" || operator == ">" || operator == ">=" }
func (c *windowCode) emitRuntimeTextComparisonOperands(left, right, condition value, slots map[string]uint32, constants map[string]constantValue) error {
	scratchBase := c.temporaryBufferOffset + runtimeTextConcatBaseScratch + runtimeTextConcatScratchStride*runtimeTextConcatScratchLevel(condition)
	if err := c.emitRuntimeTextOperand(left, slots, constants, 21, 23, scratchBase); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(21, 31, 0), storeIntegerSlot(23, 31, 8))
	if err := c.emitRuntimeTextOperand(right, slots, constants, 22, 24, scratchBase+256); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(21, 31, 0), loadIntegerSlot(23, 31, 8), addImmediate(31, 31, 16))
	return nil
}
func (c *windowCode) emitRuntimeTextOrderingFalseBranches(condition value, slots map[string]uint32, constants map[string]constantValue) ([]conditionFalsePatch, error) {
	if condition.kind != "比较" || condition.left == nil || condition.right == nil || !isTextOrderingOperator(condition.operator) {
		return nil, fmt.Errorf("运行时文本排序比较需要两个文本操作数")
	}
	falsePatches := []conditionFalsePatch{}
	afterPatches := []int{}
	if err := c.emitRuntimeTextComparisonOperands(*condition.left, *condition.right, condition, slots, constants); err != nil {
		return nil, err
	}
	c.words = append(c.words, movz(25, 0))
	loop := len(c.words)
	c.words = append(c.words, compareRegister(25, 21))
	leftEnded := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, compareRegister(25, 22))
	rightEnded := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words,
		addRegister(26, 23, 25),
		loadByte(27, 26),
		addRegister(26, 24, 25),
		loadByte(28, 26),
		compareRegister(27, 28),
	)
	equalByte := c.conditionalPlaceholder(0) // b.eq
	falseCondition := uint32(2)              // b.hs for <
	switch condition.operator {
	case "<=":
		falseCondition = 8 // b.hi
	case ">":
		falseCondition = 9 // b.ls
	case ">=":
		falseCondition = 3 // b.lo
	}
	falsePatches = append(falsePatches, conditionFalsePatch{position: c.conditionalPlaceholder(falseCondition).position, condition: falseCondition})
	trueUnequal := c.unconditionalPlaceholder()
	equalStart := len(c.words)
	c.patchConditional(equalByte, equalStart)
	c.words = append(c.words, addImmediate(25, 25, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(leftEnded, len(c.words))
	c.words = append(c.words, compareRegister(21, 22))
	switch condition.operator {
	case "<":
		patch := c.conditionalPlaceholder(0) // b.eq
		falsePatches = append(falsePatches, conditionFalsePatch{position: patch.position, condition: patch.condition})
		afterPatches = append(afterPatches, c.unconditionalPlaceholder())
	case "<=":
		afterPatches = append(afterPatches, c.unconditionalPlaceholder())
	case ">":
		falsePatches = append(falsePatches, conditionFalsePatch{position: c.unconditionalPlaceholder(), unconditional: true})
	case ">=":
		falsePatches = append(falsePatches, conditionFalsePatch{position: c.conditionalPlaceholder(3).position, condition: 3}) // b.lo
		afterPatches = append(afterPatches, c.unconditionalPlaceholder())
	}
	c.patchConditional(rightEnded, len(c.words))
	if condition.operator == "<" || condition.operator == "<=" {
		falsePatches = append(falsePatches, conditionFalsePatch{position: c.unconditionalPlaceholder(), unconditional: true})
	} else {
		afterPatches = append(afterPatches, c.unconditionalPlaceholder())
	}
	after := len(c.words)
	c.words[trueUnequal] = branch(after - trueUnequal)
	for _, position := range afterPatches {
		c.words[position] = branch(after - position)
	}
	return falsePatches, nil
}
func (c *windowCode) emitRuntimeTextComparisonFalseBranches(condition value, slots map[string]uint32, constants map[string]constantValue) ([]conditionFalsePatch, error) {
	if condition.text != "equals" && condition.text != "notEquals" || len(condition.args) != 2 {
		return nil, fmt.Errorf("运行时文本比较需要 equals/notEquals（文本，文本）")
	}
	falsePatches := []conditionFalsePatch{}
	localMismatchPatches := []conditionalPatch{}
	if err := c.emitRuntimeTextComparisonOperands(condition.args[0], condition.args[1], condition, slots, constants); err != nil {
		return nil, err
	}
	leftOwned := isTextReturnCall(condition.args[0])
	rightOwned := isTextReturnCall(condition.args[1])
	if !leftOwned && !rightOwned {
		c.words = append(c.words, compareRegister(21, 22))
		lengthMismatch := c.conditionalPlaceholder(1) // b.ne
		if condition.text == "equals" {
			falsePatches = append(falsePatches, conditionFalsePatch{position: lengthMismatch.position, condition: lengthMismatch.condition})
		} else {
			localMismatchPatches = append(localMismatchPatches, lengthMismatch)
		}
		c.words = append(c.words, movz(25, 0))
		loop := len(c.words)
		c.words = append(c.words, compareRegister(25, 21))
		finished := c.conditionalPlaceholder(10) // b.ge
		c.words = append(c.words, addRegister(26, 23, 25), loadByte(27, 26), addRegister(26, 24, 25), loadByte(28, 26), compareRegister(27, 28))
		mismatch := c.conditionalPlaceholder(1) // b.ne
		if condition.text == "equals" {
			falsePatches = append(falsePatches, conditionFalsePatch{position: mismatch.position, condition: mismatch.condition})
		} else {
			localMismatchPatches = append(localMismatchPatches, mismatch)
		}
		c.words = append(c.words, addImmediate(25, 25, 1))
		c.words = append(c.words, branch(loop-len(c.words)))
		equalEnd := len(c.words)
		c.patchConditional(finished, equalEnd)
		if condition.text == "notEquals" {
			position := c.unconditionalPlaceholder()
			falsePatches = append(falsePatches, conditionFalsePatch{position: position, unconditional: true})
		}
		differentEnd := len(c.words)
		for _, patch := range localMismatchPatches {
			c.patchConditional(patch, differentEnd)
		}
		return falsePatches, nil
	}
	return falsePatches, nil
}
func (c *windowCode) branchToRuntimeFailure(condition uint32) {
	c.failurePatches = append(c.failurePatches, c.conditionalPlaceholder(condition))
}
func (c *windowCode) emitRuntimeIntegerComparison(condition value, constants map[string]constantValue, slots map[string]uint32) (uint32, error) {
	var left, right value
	branchWhenFalse := uint32(1) // b.ne for equality
	switch condition.kind {
	case "引用":
		left, right = condition, value{kind: "整数", text: "0"}
		branchWhenFalse = 0 // b.eq for false
	case "比较":
		if condition.left == nil || condition.right == nil {
			return 0, fmt.Errorf("运行时整数比较缺少操作数")
		}
		left, right = *condition.left, *condition.right
		switch condition.operator {
		case "==":
			branchWhenFalse = 1 // b.ne
		case "!=":
			branchWhenFalse = 0 // b.eq
		case "<":
			branchWhenFalse = 10 // b.ge
		case "<=":
			branchWhenFalse = 12 // b.gt
		case ">":
			branchWhenFalse = 13 // b.le
		case ">=":
			branchWhenFalse = 11 // b.lt
		default:
			return 0, fmt.Errorf("不支持的整数比较符：%s", condition.operator)
		}
	case "调用":
		if condition.text == "arrayGet" && len(condition.args) == 2 {
			left, right, branchWhenFalse = condition, value{kind: "整数", text: "0"}, 0
		} else if condition.text == "equals" || condition.text == "notEquals" {
			if len(condition.args) != 2 {
				return 0, fmt.Errorf("运行时整数条件需要 equals、notEquals 或整数比较符")
			}
			left, right = condition.args[0], condition.args[1]
			if condition.text == "notEquals" {
				branchWhenFalse = 0 // b.eq
			}
		} else {
			return 0, fmt.Errorf("运行时整数条件需要 equals、notEquals 或整数比较符")
		}
	default:
		return 0, fmt.Errorf("不支持的运行时整数条件：%s", condition.kind)
	}
	if err := c.emitRuntimeInteger(left, constants, slots); err != nil {
		return 0, err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), 0xf90003e0)
	if err := c.emitRuntimeInteger(right, constants, slots); err != nil {
		return 0, err
	}
	c.words = append(c.words, 0xaa0003e1, 0xf94003e0, addImmediate(31, 31, 16), compareRegister(0, 1))
	return branchWhenFalse, nil
}
func (c *windowCode) emitRuntimeFailureBlock(exitIndex int) {
	if len(c.failurePatches) == 0 {
		return
	}
	failure := len(c.words)
	for _, patch := range c.failurePatches {
		c.patchConditional(patch, failure)
	}
	c.movImmediate(0, 1)
	c.call(exitIndex) // _exit(1)
	c.failurePatches = nil
}
func (c *windowCode) emitConsoleString(text string, writeIndex, exitIndex int) error {
	output := append([]byte(text), '\n')
	c.addData("consoleOutput", output, 1)
	c.movImmediate(0, 1) // stdout
	c.address("consoleOutput", 1)
	if err := c.emitRegisterImmediate(2, int64(len(output))); err != nil {
		return err
	}
	c.call(writeIndex) // write(stdout, UTF-8 literal, byte length)
	c.movImmediate(0, 0)
	c.call(exitIndex) // _exit(0)
	return nil
}
func (c *windowCode) emitReadInteger(bufferOffset uint32) error {
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(20, 31, 0))
	c.emitConsoleStackAddress(21, bufferOffset)
	c.movImmediate(20, 0)
	readLoop := len(c.words)
	c.words = append(c.words, compareImmediate(20, 64))
	bufferFull := c.conditionalPlaceholder(10) // b.ge
	c.movImmediate(0, 0)
	c.words = append(c.words, addRegister(1, 21, 20))
	c.movImmediate(2, 1)
	c.call(c.fileReadIndex) // read one byte for consistent terminal and pipe behavior
	c.words = append(c.words, compareImmediate(0, 0))
	readFailure := c.conditionalPlaceholder(11) // b.lt
	readEOF := c.conditionalPlaceholder(0)      // b.eq
	c.words = append(c.words, compareImmediate(0, 1))
	readShort := c.conditionalPlaceholder(1) // b.ne
	c.words = append(c.words, addImmediate(20, 20, 1), addRegister(22, 21, 20), subImmediate(22, 22, 1), loadByte(23, 22), compareWordImmediate(23, 10))
	lineComplete := c.conditionalPlaceholder(0) // b.eq
	readLoopBack := c.unconditionalPlaceholder()
	endOfInput := len(c.words)
	c.words = append(c.words, compareImmediate(20, 0))
	emptyInput := c.conditionalPlaceholder(0) // b.eq
	endOfInputToParse := c.unconditionalPlaceholder()
	parseLine := len(c.words)
	c.words[readLoopBack] = branch(readLoop - readLoopBack)
	c.patchConditional(lineComplete, parseLine)
	c.words = append(c.words, subImmediate(20, 20, 1), compareImmediate(20, 0))
	emptyAfterLineFeed := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, addRegister(22, 21, 20), subImmediate(22, 22, 1), loadByte(23, 22), compareWordImmediate(23, 13))
	withoutCarriageReturn := c.conditionalPlaceholder(1) // b.ne
	c.words = append(c.words, subImmediate(20, 20, 1), compareImmediate(20, 0))
	emptyAfterCarriageReturn := c.conditionalPlaceholder(0) // b.eq
	parseSign := len(c.words)
	c.words[endOfInputToParse] = branch(parseSign - endOfInputToParse)
	c.movImmediate(23, 0)
	c.movImmediate(24, 0)
	c.words = append(c.words, loadByte(25, 21), compareWordImmediate(25, 45))
	withoutMinus := c.conditionalPlaceholder(1) // b.ne
	c.movImmediate(24, 1)
	c.words = append(c.words, addImmediate(21, 21, 1), subImmediate(20, 20, 1), compareImmediate(20, 0))
	emptyAfterMinus := c.conditionalPlaceholder(0) // b.eq
	parseDigits := len(c.words)
	c.words = append(c.words, compareImmediate(20, 0))
	finishedDigits := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, loadByte(25, 21), compareWordImmediate(25, 48))
	digitTooSmall := c.conditionalPlaceholder(11) // b.lt
	c.words = append(c.words, compareWordImmediate(25, 57))
	digitTooLarge := c.conditionalPlaceholder(12) // b.gt
	c.words = append(c.words, subtractWordImmediate(25, 25, 48))
	if err := c.emitRegisterImmediate(27, -922337203685477580); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(23, 27))
	accumulatorTooSmall := c.conditionalPlaceholder(11) // b.lt
	accumulatorSafe := c.conditionalPlaceholder(12)     // b.gt
	c.words = append(c.words, compareImmediate(24, 0))
	negativeLimit := c.conditionalPlaceholder(1) // b.ne
	c.words = append(c.words, compareWordImmediate(25, 7))
	positiveDigitTooLarge := c.conditionalPlaceholder(12) // b.gt
	positiveLimitSafe := c.unconditionalPlaceholder()
	checkNegativeLimit := len(c.words)
	c.words = append(c.words, compareWordImmediate(25, 8))
	negativeDigitTooLarge := c.conditionalPlaceholder(12) // b.gt
	accumulate := len(c.words)
	if err := c.emitRegisterImmediate(26, 10); err != nil {
		return err
	}
	c.words = append(c.words, multiplyRegister(23, 23, 26), subtractRegister(23, 23, 25), addImmediate(21, 21, 1), subImmediate(20, 20, 1))
	digitLoopBack := c.unconditionalPlaceholder()
	finish := len(c.words)
	c.words = append(c.words, compareImmediate(24, 0))
	negativeResult := c.conditionalPlaceholder(1) // b.ne
	c.words = append(c.words, 0xcb1703e0)         // neg x0, x23
	resultReady := c.unconditionalPlaceholder()
	negativeResultTarget := len(c.words)
	c.movRegister(0, 23)
	resultDone := c.unconditionalPlaceholder()
	failure := len(c.words)
	c.movImmediate(0, 1)
	c.call(c.exitIndex) // _exit(1)
	afterFailure := len(c.words)
	c.words = append(c.words, loadIntegerSlot(20, 31, 0), addImmediate(31, 31, 16))
	for _, patch := range []conditionalPatch{bufferFull, readFailure, readShort, emptyInput, emptyAfterLineFeed, emptyAfterCarriageReturn, emptyAfterMinus, digitTooSmall, digitTooLarge, accumulatorTooSmall, positiveDigitTooLarge, negativeDigitTooLarge} {
		c.patchConditional(patch, failure)
	}
	c.patchConditional(readEOF, endOfInput)
	c.patchConditional(withoutCarriageReturn, parseSign)
	c.patchConditional(withoutMinus, parseDigits)
	c.patchConditional(finishedDigits, finish)
	c.patchConditional(accumulatorSafe, accumulate)
	c.patchConditional(negativeLimit, checkNegativeLimit)
	c.patchConditional(negativeResult, negativeResultTarget)
	c.words[positiveLimitSafe] = branch(accumulate - positiveLimitSafe)
	c.words[digitLoopBack] = branch(parseDigits - digitLoopBack)
	c.words[resultReady] = branch(afterFailure - resultReady)
	c.words[resultDone] = branch(afterFailure - resultDone)
	return nil
}
func (c *windowCode) emitReadText(bufferOffset uint32, storedLengthOffset *uint32, retainLength bool, readIndex, writeIndex, exitIndex int) error {
	c.movRegister(28, 20)
	c.emitConsoleStackAddress(21, bufferOffset)
	c.movImmediate(20, 0)
	readLoop := len(c.words)
	c.words = append(c.words, compareImmediate(20, 63))
	bufferExceeded := c.conditionalPlaceholder(12) // b.gt
	c.failurePatches = append(c.failurePatches, bufferExceeded)
	c.movImmediate(0, 0)
	c.words = append(c.words, addRegister(1, 21, 20))
	c.movImmediate(2, 1)
	c.call(readIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	readFailure := c.conditionalPlaceholder(11) // b.lt
	readEOF := c.conditionalPlaceholder(0)      // b.eq
	c.words = append(c.words, addRegister(24, 21, 20), loadByte(23, 24), compareWordImmediate(23, 10))
	lineComplete := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, compareImmediate(20, 63))
	lastByte := c.conditionalPlaceholder(0) // b.eq; only a line terminator fits at byte 63
	c.failurePatches = append(c.failurePatches, lastByte)
	c.words = append(c.words, addImmediate(20, 20, 1), branch(readLoop-len(c.words)-1))
	finish := len(c.words)
	c.words = append(c.words, compareImmediate(20, 0))
	empty := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, subImmediate(22, 20, 1), addRegister(24, 21, 22), loadByte(23, 24), compareWordImmediate(23, 13))
	withoutCarriageReturn := c.conditionalPlaceholder(1) // b.ne
	c.words = append(c.words, subImmediate(20, 20, 1))
	validate := len(c.words)
	c.failurePatches = append(c.failurePatches, readFailure)
	c.patchConditional(readEOF, finish)
	c.patchConditional(lineComplete, finish)
	c.patchConditional(empty, len(c.words))
	c.patchConditional(withoutCarriageReturn, validate)
	c.emitRuntimeUTF8Validation(21, 20)
	c.words = append(c.words, addRegister(24, 21, 20))
	c.movImmediate(23, 10)
	c.words = append(c.words, 0x39000000|24<<5|23) // strb w23, [x24]
	if storedLengthOffset != nil {
		c.words = append(c.words, storeIntegerSlot(20, 19, *storedLengthOffset))
	}
	if storedLengthOffset != nil || retainLength {
		c.movRegister(22, 20)
		c.movRegister(20, 28)
		return nil
	}
	c.movImmediate(0, 1)
	c.movRegister(1, 21)
	c.words = append(c.words, addImmediate(2, 20, 1))
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
	return nil
}
func (c *windowCode) emitReadDynamicText(pointer, length, capacity uint32, readIndex, exitIndex int) error {
	const maximum = 16 * 1024 * 1024
	c.words = append(c.words, subImmediate(31, 31, 64), storeIntegerSlot(20, 31, 56), loadIntegerSlot(21, 19, pointer), loadIntegerSlot(22, 19, capacity), movz(20, 0), storeIntegerSlot(20, 19, length))
	readLoop := len(c.words)
	c.words = append(c.words, compareRegister(20, 22))
	needsGrowth := c.conditionalPlaceholder(2) // b.hs
	readCurrent := c.unconditionalPlaceholder()
	grow := len(c.words)
	c.patchConditional(needsGrowth, grow)
	c.words = append(c.words, storeIntegerSlot(21, 31, 0), storeIntegerSlot(22, 31, 8), storeIntegerSlot(20, 31, 16))
	if err := c.emitRegisterImmediate(4, maximum); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(22, 4))
	c.branchToRuntimeFailure(2) // b.hs at the configured line-size limit
	if err := c.emitRegisterImmediate(4, maximum/2); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(22, 4))
	newCapacityIsMaximum := c.conditionalPlaceholder(12) // b.gt
	c.words = append(c.words, logicalShiftLeft(24, 22, 1), storeIntegerSlot(24, 31, 24))
	capacityReady := c.unconditionalPlaceholder()
	c.patchConditional(newCapacityIsMaximum, len(c.words))
	if err := c.emitRegisterImmediate(24, maximum); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(24, 31, 24))
	c.words[capacityReady] = branch(len(c.words) - capacityReady)
	if err := c.emitArrayMapCall(24); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 32), movz(7, 0))
	copyLoop := len(c.words)
	c.words = append(c.words, loadIntegerSlot(1, 31, 16), compareRegister(7, 1))
	copyDone := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, loadIntegerSlot(1, 31, 0), addRegister(2, 1, 7), loadByte(3, 2), loadIntegerSlot(1, 31, 32), addRegister(2, 1, 7), storeByte(3, 2), addImmediate(7, 7, 1))
	c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), loadIntegerSlot(1, 31, 8))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1) // b.ne
	c.words = append(c.words, loadIntegerSlot(21, 31, 32), loadIntegerSlot(22, 31, 24), loadIntegerSlot(20, 31, 16), storeIntegerSlot(21, 19, pointer), storeIntegerSlot(22, 19, capacity))
	readCurrentLabel := len(c.words)
	c.words[readCurrent] = branch(readCurrentLabel - readCurrent)
	c.movImmediate(0, 0)
	c.words = append(c.words, addRegister(1, 21, 20))
	c.movImmediate(2, 1)
	c.call(readIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	readFailure := c.conditionalPlaceholder(11) // b.lt
	readEOF := c.conditionalPlaceholder(0)      // b.eq
	c.words = append(c.words, addRegister(24, 21, 20), loadByte(23, 24), compareWordImmediate(23, 10))
	lineComplete := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, addImmediate(20, 20, 1), storeIntegerSlot(20, 19, length))
	c.words = append(c.words, branch(readLoop-len(c.words)))
	finish := len(c.words)
	c.failurePatches = append(c.failurePatches, readFailure)
	c.patchConditional(readEOF, finish)
	c.patchConditional(lineComplete, finish)
	c.words = append(c.words, compareImmediate(20, 0))
	empty := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, subImmediate(23, 20, 1), addRegister(24, 21, 23), loadByte(25, 24), compareWordImmediate(25, 13))
	withoutCarriageReturn := c.conditionalPlaceholder(1) // b.ne
	c.words = append(c.words, subImmediate(20, 20, 1))
	validate := len(c.words)
	c.patchConditional(empty, validate)
	c.patchConditional(withoutCarriageReturn, validate)
	c.emitRuntimeUTF8Validation(21, 20)
	c.words = append(c.words, storeIntegerSlot(20, 19, length), addRegister(24, 21, 20))
	c.movImmediate(23, 10)
	c.words = append(c.words, storeByte(23, 24), loadIntegerSlot(20, 31, 56), addImmediate(31, 31, 64))
	return nil
}
func (c *windowCode) emitRuntimeTextFunctionOutput(input value, constants map[string]constantValue, slots map[string]uint32, writeIndex, exitIndex int) error {
	if !input.runtimeFunctionText {
		return fmt.Errorf("运行时文本输出需要文本返回函数调用")
	}
	if err := c.emitRuntimeFunctionCall(input, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(1, 31, 0), storeIntegerSlot(3, 31, 8))
	c.movRegister(21, 1)
	c.words = append(c.words, addImmediate(22, 2, 1), movz(23, 0))
	writeLoop := len(c.words)
	c.words = append(c.words, movz(0, 1))
	c.movRegister(1, 21)
	c.movRegister(2, 22)
	c.call(writeIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	writeFailed := c.conditionalPlaceholder(13) // b.le
	c.words = append(c.words, addRegister(21, 21, 0), subtractRegister(22, 22, 0), compareImmediate(22, 0))
	c.words = append(c.words, branchCondition(writeLoop-len(c.words), 1))
	writeDone := c.unconditionalPlaceholder()
	c.patchConditional(writeFailed, len(c.words))
	c.words = append(c.words, movz(23, 1))
	c.words[writeDone] = branch(len(c.words) - writeDone)
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), loadIntegerSlot(1, 31, 8), addImmediate(31, 31, 16))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.movRegister(0, 23)
	c.call(exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeResultTextOutput(input value, constants map[string]constantValue, slots map[string]uint32, writeIndex, exitIndex int) error {
	if !isDirectTextResultExtraction(input) {
		return fmt.Errorf("运行时文本结果输出需要直接文本结果值")
	}
	result := input.args[0]
	if err := c.emitResultValueRegisters(result, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(0, 31, 0), storeIntegerSlot(1, 31, 8), storeIntegerSlot(2, 31, 16), storeIntegerSlot(3, 31, 24))
	c.words = append(c.words, compareImmediate(0, 1))
	success := c.conditionalPlaceholder(1)
	c.movRegister(0, 1)
	c.call(exitIndex)
	c.patchConditional(success, len(c.words))
	c.words = append(c.words, loadIntegerSlot(21, 31, 16), loadIntegerSlot(22, 31, 24), addImmediate(22, 22, 1), movz(23, 0))
	writeLoop := len(c.words)
	c.words = append(c.words, movz(0, 1))
	c.movRegister(1, 21)
	c.movRegister(2, 22)
	c.call(writeIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	writeFailed := c.conditionalPlaceholder(13)
	c.words = append(c.words, addRegister(21, 21, 0), subtractRegister(22, 22, 0), compareImmediate(22, 0))
	c.words = append(c.words, branchCondition(writeLoop-len(c.words), 1))
	writeDone := c.unconditionalPlaceholder()
	c.patchConditional(writeFailed, len(c.words))
	c.movImmediate(23, 1)
	c.words[writeDone] = branch(len(c.words) - writeDone)
	if runtimeResultPayload(result, constants) == "运行时文本" {
		c.words = append(c.words, loadIntegerSlot(0, 31, 16), loadIntegerSlot(1, 31, 8))
		c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		c.branchToRuntimeFailure(1)
	}
	c.words = append(c.words, addImmediate(31, 31, 32))
	c.movRegister(0, 23)
	c.call(exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeTextOutput(bufferOffset, lengthOffset uint32, writeIndex, exitIndex int) {
	c.words = append(c.words, loadIntegerSlot(2, 19, lengthOffset), addImmediate(2, 2, 1))
	c.movImmediate(0, 1)
	c.emitConsoleStackAddress(1, bufferOffset)
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
}
func (c *windowCode) emitRuntimeTextOutputPointer(pointerOffset, lengthOffset uint32, writeIndex, exitIndex int) {
	c.words = append(c.words, loadIntegerSlot(2, 19, lengthOffset), addImmediate(2, 2, 1))
	c.movImmediate(0, 1)
	c.words = append(c.words, loadIntegerSlot(1, 19, pointerOffset))
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
}
func (c *windowCode) emitRuntimeTextOutputRegister(bufferOffset, lengthRegister uint32, writeIndex, exitIndex int) {
	c.movRegister(2, lengthRegister)
	c.words = append(c.words, addImmediate(2, 2, 1))
	c.movImmediate(0, 1)
	c.emitConsoleStackAddress(1, bufferOffset)
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
}
func (c *windowCode) emitRuntimeArgumentTextOutput(writeIndex int) error {
	c.movRegister(20, 22)
	c.movImmediate(0, 1)
	c.movRegister(1, 21)
	c.movRegister(2, 20)
	c.call(writeIndex)
	label := fmt.Sprintf("runtimeArgumentNewline%d", len(c.dataLabels))
	c.addData(label, []byte{'\n'}, 1)
	c.address(label, 1)
	c.movImmediate(0, 1)
	c.movImmediate(2, 1)
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(c.exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeFileSize(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeFileSize(input) || c.fileOpenIndex < 0 || c.moduleStatIndex < 0 {
		return fmt.Errorf("fileSize 需要静态或参数文本路径和 _open/_fstat/_close")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 25, 26); err != nil {
		return err
	}
	c.movRegister(0, 26)
	c.movImmediate(1, 0) // O_RDONLY
	c.movImmediate(2, 0)
	c.call(c.fileOpenIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	openReady := c.conditionalPlaceholder(10) // b.ge
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(openReady, len(c.words))
	c.movRegister(22, 0)
	c.words = append(c.words, subImmediate(31, 31, 160), storeIntegerSlot(0, 31, 0))
	c.words = append(c.words, addImmediate(1, 31, 32), addRegister(2, 31, 32))
	c.call(c.moduleStatIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	statReady := c.conditionalPlaceholder(10)
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(statReady, len(c.words))
	c.words = append(c.words, loadIntegerSlot(22, 31, 0))
	c.movRegister(0, 22)
	c.call(c.fileCloseIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	closeReady := c.conditionalPlaceholder(10)
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(closeReady, len(c.words))
	c.words = append(c.words, addImmediate(1, 31, 32+96), loadIntegerSlot(0, 1, 0), addImmediate(31, 31, 160))
	return nil
}
func (c *windowCode) emitRuntimePathExists(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimePathExists(input) || c.fileOpenIndex < 0 || c.moduleStatIndex < 0 || c.fileCloseIndex < 0 {
		return fmt.Errorf("exists 需要静态或参数文本路径和 _open/_fstat/_close")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 25, 26); err != nil {
		return err
	}
	c.movRegister(0, 26)
	c.movImmediate(1, 0) // O_RDONLY
	c.movImmediate(2, 0)
	c.call(c.fileOpenIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	openReady := c.conditionalPlaceholder(10) // b.ge
	missing := len(c.words)
	c.words = append(c.words, branch(0))
	c.patchConditional(openReady, len(c.words))
	c.movRegister(22, 0)
	c.words = append(c.words, subImmediate(31, 31, 160), storeIntegerSlot(0, 31, 0), addImmediate(1, 31, 32), addRegister(2, 31, 32))
	c.call(c.moduleStatIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	statReady := c.conditionalPlaceholder(10)
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(statReady, len(c.words))
	c.words = append(c.words, loadIntegerSlot(22, 31, 0))
	c.movRegister(0, 22)
	c.call(c.fileCloseIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	closeReady := c.conditionalPlaceholder(10)
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(closeReady, len(c.words))
	c.movImmediate(0, 1)
	found := len(c.words)
	c.words = append(c.words, branch(0))
	c.words[missing] = branch(len(c.words) - missing)
	c.movImmediate(0, 0)
	c.words[found] = branch(len(c.words) - found)
	c.words = append(c.words, addImmediate(31, 31, 160))
	return nil
}
func (c *windowCode) emitRuntimeDirectoryCheck(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeDirectoryCheck(input) || c.fileOpenIndex < 0 || c.moduleStatIndex < 0 || c.fileCloseIndex < 0 {
		return fmt.Errorf("目录类型查询需要 isDirectory（静态或参数文本路径）和 _open/_fstat/_close")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 25, 26); err != nil {
		return err
	}
	c.movRegister(0, 26)
	c.movImmediate(1, 0)
	c.movImmediate(2, 0)
	c.call(c.fileOpenIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	openReady := c.conditionalPlaceholder(10)
	missing := len(c.words)
	c.words = append(c.words, branch(0))
	c.patchConditional(openReady, len(c.words))
	c.movRegister(22, 0)
	c.words = append(c.words, subImmediate(31, 31, 160), storeIntegerSlot(0, 31, 0), addImmediate(1, 31, 32), addRegister(2, 31, 32))
	c.call(c.moduleStatIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	statReady := c.conditionalPlaceholder(10)
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(statReady, len(c.words))
	c.words = append(c.words, loadIntegerSlot(22, 31, 0))
	c.movRegister(0, 22)
	c.call(c.fileCloseIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	closeReady := c.conditionalPlaceholder(10)
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(closeReady, len(c.words))
	// Darwin st_mode is a uint16 at stat+4; S_IFDIR is bit 0x4000.
	c.movImmediate(0, 0)
	c.words = append(c.words, addImmediate(24, 31, 37), loadByte(25, 24))
	c.movImmediate(26, 0x40)
	c.words = append(c.words, andRegister(25, 25, 26), compareImmediate(25, 0))
	notDirectory := c.conditionalPlaceholder(0) // b.eq
	c.movImmediate(0, 1)
	found := len(c.words)
	c.words = append(c.words, branch(0))
	c.words[missing] = branch(len(c.words) - missing)
	c.patchConditional(notDirectory, len(c.words))
	c.movImmediate(0, 0)
	c.words[found] = branch(len(c.words) - found)
	return nil
}
func (c *windowCode) emitRuntimeFileWrite(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeFileWrite(input) {
		return fmt.Errorf("文件写入需要 writeFile/appendFile（静态或参数文本路径，文本）")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 26, 27); err != nil {
		return err
	}
	byteArray := input.text == "writeByteArray"
	if byteArray {
		if err := c.packRuntimeBytes(input.args[1]); err != nil {
			return err
		}
	} else {
		c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(27, 31, 0))
		if err := c.emitRuntimeTextOperand(input.args[1], slots, constants, 22, 21, c.temporaryBufferOffset+256); err != nil {
			return err
		}
		c.words = append(c.words, loadIntegerSlot(27, 31, 0), addImmediate(31, 31, 16))
	}
	c.movRegister(0, 27)
	c.movImmediate(1, map[bool]uint16{false: 0644, true: 521}[input.text == "appendFile"])
	c.words = append(c.words, subImmediate(31, 31, 16), movz(2, 0644), storeIntegerSlot(2, 31, 0))
	c.call(map[bool]int{false: c.fileCreateIndex, true: c.fileOpenIndex}[input.text == "appendFile"])
	c.words = append(c.words, addImmediate(31, 31, 16))
	c.words = append(c.words, compareImmediate(0, 0))
	openReady := c.conditionalPlaceholder(10) // b.ge
	c.movImmediate(0, 21)
	c.call(c.exitIndex)
	c.patchConditional(openReady, len(c.words))
	c.movRegister(23, 0)
	c.movImmediate(24, 0)
	writeLoop := len(c.words)
	c.words = append(c.words, compareRegister(24, 22))
	writeDone := c.conditionalPlaceholder(10) // b.ge
	c.movRegister(0, 23)
	c.words = append(c.words, addRegister(1, 21, 24), subtractRegister(2, 22, 24))
	c.call(c.writeIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	writeProgress := c.conditionalPlaceholder(12) // b.gt
	c.movImmediate(0, 22)
	c.call(c.exitIndex)
	c.patchConditional(writeProgress, len(c.words))
	c.words = append(c.words, addRegister(24, 24, 0), branch(writeLoop-len(c.words)-1))
	c.patchConditional(writeDone, len(c.words))
	c.movRegister(0, 23)
	c.call(c.fileCloseIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	closeReady := c.conditionalPlaceholder(0) // b.eq
	c.movImmediate(0, 23)
	c.call(c.exitIndex)
	c.patchConditional(closeReady, len(c.words))
	if byteArray {
		c.movRegister(0, 21)
		c.movRegister(1, 22)
		c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		c.branchToRuntimeFailure(1)
	}
	c.movRegister(0, 22)
	return nil
}
func (c *windowCode) emitRuntimeFileRemove(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeFileRemove(input) || c.fileRemoveIndex < 0 {
		return fmt.Errorf("文件删除需要 removeFile（静态或参数文本路径）")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 25, 26); err != nil {
		return err
	}
	c.movImmediate(24, 1); c.movRegister(0, 26); c.call(c.fileRemoveIndex)
	c.emitRuntimeFileMutationResult(c.exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeFileRename(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeFileRename(input) || c.fileRenameIndex < 0 {
		return fmt.Errorf("文件重命名需要 renameFile（静态或参数文本路径，静态或参数文本路径）")
	}
	return c.emitRuntimeFilePathPair(input, constants, slots, "文件重命名")
}
func (c *windowCode) emitRuntimeFileReplace(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeFileReplace(input) || c.fileRenameIndex < 0 {
		return fmt.Errorf("文件原子替换需要 replaceFile（静态或参数文本路径，静态或参数文本路径）")
	}
	return c.emitRuntimeFilePathPair(input, constants, slots, "文件原子替换")
}
func (c *windowCode) emitRuntimeFilePathPair(input value, constants map[string]constantValue, slots map[string]uint32, label string) error {
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 24, 25); err != nil {
		return fmt.Errorf("%s源路径：%w", label, err)
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(25, 31, 0))
	if err := c.emitRuntimeFilePath(input.args[1], constants, slots, 26, 27); err != nil {
		return fmt.Errorf("%s目标路径：%w", label, err)
	}
	c.words = append(c.words, loadIntegerSlot(0, 31, 0)); c.movRegister(1, 27)
	c.words = append(c.words, addImmediate(31, 31, 16)); c.movImmediate(24, 1); c.call(c.fileRenameIndex)
	c.emitRuntimeFileMutationResult(c.exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeDirectoryMutation(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeDirectoryMutation(input) || c.fileMkdirIndex < 0 && c.fileRmdirIndex < 0 {
		return fmt.Errorf("目录操作需要 createDirectory/removeDirectory（静态或参数文本路径）")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 25, 26); err != nil {
		return err
	}
	c.movImmediate(24, 1); c.movRegister(0, 26)
	if input.text == "createDirectory" {
		c.movImmediate(1, 0755)
		c.call(c.fileMkdirIndex)
	} else {
		c.call(c.fileRmdirIndex)
	}
	c.emitRuntimeFileMutationResult(c.exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeFileMode(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if !isRuntimeFileMode(input) || input.text == "setFileMode" && c.fileChmodIndex < 0 || input.text == "changeFileMode" && c.fileFchmodIndex < 0 {
		return fmt.Errorf("权限设置需要 setFileMode/changeFileMode（静态或参数文本路径，整数权限）")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 25, 26); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(26, 31, 0), storeIntegerSlot(27, 31, 0), storeIntegerSlot(0, 31, 8))
	if input.text == "changeFileMode" {
		c.movRegister(0, 26)
		c.movImmediate(1, 0)
		c.movImmediate(2, 0)
		c.call(c.fileOpenIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		openReady := c.conditionalPlaceholder(10)
		c.movImmediate(0, 21)
		c.call(c.exitIndex)
		c.patchConditional(openReady, len(c.words))
		c.movRegister(22, 0)
	}
	if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(1, 31, 0), andRegister(1, 0, 1))
	c.words = append(c.words, loadIntegerSlot(0, 31, 8))
	if input.text == "setFileMode" {
		c.words = append(c.words, addImmediate(31, 31, 16), movz(24, 1))
		c.call(c.fileChmodIndex)
	} else {
		c.movRegister(0, 22)
		c.words = append(c.words, addImmediate(31, 31, 16), movz(24, 1))
		c.call(c.fileFchmodIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		fchmodReady := c.conditionalPlaceholder(0)
		c.movRegister(0, 21)
		c.call(c.exitIndex)
		c.patchConditional(fchmodReady, len(c.words))
		c.movRegister(0, 22)
		c.call(c.fileCloseIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		closeReady := c.conditionalPlaceholder(0)
		c.movRegister(0, 21)
		c.call(c.exitIndex)
		c.patchConditional(closeReady, len(c.words))
	}
	c.emitRuntimeFileMutationResult(c.exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeFileMutationResult(exitIndex int) {
	c.words = append(c.words, compareImmediate(0, 0))
	success := c.conditionalPlaceholder(0)
	c.movRegister(0, 24)
	c.call(exitIndex)
	c.patchConditional(success, len(c.words))
	c.movImmediate(0, 0)
}
func (c *windowCode) packRuntimeBytes(input value) error {
	array, exists := c.arraySlots[input.text]
	if !exists || array.text {
		return fmt.Errorf("writeByteArray 需要运行时整数数组")
	}
	c.words = append(c.words, loadIntegerSlot(22, 19, array.length))
	if err := c.emitArrayMapCall(22); err != nil {
		return err
	}
	c.movRegister(21, 0)
	c.words = append(c.words, loadIntegerSlot(23, 19, array.data), movz(24, 0))
	copyLoop := len(c.words)
	c.words = append(c.words, compareRegister(24, 22))
	copyDone := c.conditionalPlaceholder(10)
	c.words = append(c.words, loadIndexedRegister(25, 23, 24), compareImmediate(25, 255))
	c.branchToRuntimeFailure(8) // b.hi rejects negative integers as well.
	c.words = append(c.words, addRegister(26, 21, 24), storeByte(25, 26), addImmediate(24, 24, 1))
	c.words = append(c.words, branch(copyLoop-len(c.words)))
	c.patchConditional(copyDone, len(c.words))
	return nil
}
func (c *windowCode) emitRuntimeEnvironmentNamesTextOutput(input value, constants map[string]constantValue, slots map[string]uint32, writeIndex, exitIndex int) error {
	if !isRuntimeEnvironmentNames(input) {
		return fmt.Errorf("环境变量名称枚举需要 environmentNames（）")
	}
	// Read the NULL-terminated envp vector from the bound _environ symbol. Copy
	// up to 256 names into the scratch frame in platform enumeration order.
	const (
		namesLimit  = 256
		regCount    = 24
		regScan     = 14
		regName     = 28
		regIndex    = 21
		regSlot     = 6
		regByte     = 9
	)
	arrayOffset := c.temporaryBufferOffset + 320
	newlineSlot := c.temporaryBufferOffset + 16
	countSlot := c.temporaryBufferOffset + 256*8 + 320
	indexSlot := countSlot + 8
	c.words = append(c.words, movz(regCount, 0), storeIntegerSlot(regCount, 19, countSlot))
	// _environ is a bound data symbol: its GOT slot holds the address of the
	// envp vector, so dereference it after dyld fixups.
	c.environmentGOTAddress(regScan)
	c.words = append(c.words, loadPointer(regScan, regScan, 0), loadPointer(regScan, regScan, 0), compareImmediate(regScan, 0))
	collect := len(c.words)
	noEnviron := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadPointer(regName, regScan, 0), compareImmediate(regName, 0))
	noName := c.conditionalPlaceholder(0)
	c.words = append(c.words, compareImmediate(regCount, namesLimit))
	full := c.conditionalPlaceholder(2)
	c.words = append(c.words, addRegisterShifted(regSlot, 19, regCount, 3), addImmediate(regSlot, regSlot, arrayOffset), storeIntegerSlot(regName, regSlot, 0))
	c.words = append(c.words, addImmediate(regCount, regCount, 1), storeIntegerSlot(regCount, 19, countSlot))
	c.words = append(c.words, addImmediate(regScan, regScan, 8), branch(collect-len(c.words)))
	collected := len(c.words)
	c.patchConditional(noEnviron, collected)
	c.patchConditional(noName, collected)
	c.patchConditional(full, collected)
	c.words = append(c.words, loadIntegerSlot(regCount, 19, countSlot))

	c.words = append(c.words, movz(regIndex, 0), movz(9, 10), addImmediate(regSlot, 19, newlineSlot), storeByte(9, regSlot))
	c.words = append(c.words, storeIntegerSlot(regIndex, 19, indexSlot))
	c.movImmediate(0, 1)
	emit := len(c.words)
	c.words = append(c.words, loadIntegerSlot(regIndex, 19, indexSlot), loadIntegerSlot(regCount, 19, countSlot))
	c.words = append(c.words, compareRegister(regIndex, regCount))
	finished := c.conditionalPlaceholder(10)
	c.words = append(c.words, addRegisterShifted(regSlot, 19, regIndex, 3), addImmediate(regSlot, regSlot, arrayOffset), loadIntegerSlot(26, regSlot, 0), movz(2, 0))
	c.movRegister(1, 26)
	c.words = append(c.words, movz(2, 0))
	nameLoop := len(c.words)
	c.words = append(c.words, addRegister(regScan, 26, 2), loadByte(regByte, regScan), compareWordImmediate(regByte, 61))
	lengthDone := c.conditionalPlaceholder(0)
	c.words = append(c.words, compareWordImmediate(regByte, 0))
	zeroDone := c.conditionalPlaceholder(0)
	c.words = append(c.words, addImmediate(2, 2, 1), branch(nameLoop-len(c.words)-1))
	c.patchConditional(lengthDone, len(c.words))
	c.patchConditional(zeroDone, len(c.words))
	c.movRegister(27, 2)
	c.movRegister(26, 1)
	c.movImmediate(0, 1)
	c.movRegister(1, 26)
	c.movRegister(2, 27)
	c.call(writeIndex)
	c.words = append(c.words, movz(1, 0), movz(2, 1), addImmediate(1, 19, newlineSlot))
	c.movImmediate(0, 1)
	c.call(writeIndex)
	c.words = append(c.words, addImmediate(regIndex, regIndex, 1), storeIntegerSlot(regIndex, 19, indexSlot), branch(emit-len(c.words)))
	c.patchConditional(finished, len(c.words))
	c.movImmediate(0, 0)
	c.call(exitIndex)
	return nil
}

func (c *windowCode) emitRuntimeEnvironmentNames(input value, constants map[string]constantValue, slots map[string]uint32) (uint32, uint32, error) {
	if !isRuntimeEnvironmentNames(input) || c.environmentIndex < 0 {
		return 0, 0, fmt.Errorf("环境变量名称枚举需要 environmentNames（）和 _environ")
	}
	return 0, 0, fmt.Errorf("环境变量名称枚举暂不支持作为容器载荷")
}
func (c *windowCode) emitRuntimeEnvironmentRead(input value, constants map[string]constantValue, slots map[string]uint32) (uint32, uint32, error) {
	if !isRuntimeEnvironmentRead(input) || c.getenvIndex < 0 {
		return 0, 0, fmt.Errorf("环境变量查询需要 readEnvironment（静态文本或参数文本名称）")
	}
	if isRuntimeArgumentText(input.args[0]) {
		return 0, 0, fmt.Errorf("动态环境变量名称尚不支持")
	}
	name, problem := evaluateString(input.args[0], 0, constants)
	if problem != nil {
		return 0, 0, fmt.Errorf("环境变量名称无法求值：%v", problem)
	}
	label := fmt.Sprintf("runtimeEnvironmentName%d", len(c.dataLabels))
	c.addData(label, append([]byte(name), 0), 1)
	c.address(label, 0)
	c.call(c.getenvIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	notFound := c.conditionalPlaceholder(0)
	c.movImmediate(21, 0)
	lengthLoop := len(c.words)
	c.words = append(c.words, addRegister(25, 0, 21), loadByte(25, 25), compareImmediate(25, 0))
	lengthDone := c.conditionalPlaceholder(0)
	c.words = append(c.words, addImmediate(21, 21, 1), branch(lengthLoop-len(c.words)-1))
	c.patchConditional(lengthDone, len(c.words))
	c.emitRuntimeUTF8Validation(0, 21)
	valid := c.unconditionalPlaceholder()
	c.patchConditional(notFound, len(c.words))
	emptyLabel := fmt.Sprintf("runtimeEnvironmentEmpty%d", len(c.dataLabels))
	c.addData(emptyLabel, []byte{0}, 1)
	c.address(emptyLabel, 0)
	c.words = append(c.words, movz(21, 0))
	c.words[valid] = branch(len(c.words) - valid)
	return 0, 21, nil
}
func (c *windowCode) emitRuntimeFilePath(input value, constants map[string]constantValue, slots map[string]uint32, lengthRegister, pointerRegister uint32) error {
	if path, problem := evaluateString(input, 0, constants); problem == nil {
		if path == "" || strings.IndexByte(path, 0) >= 0 {
			return fmt.Errorf("静态文件路径不能为空或包含 NUL")
		}
		label := fmt.Sprintf("runtimeFilePath%d", len(c.dataLabels))
		c.addData(label, append([]byte(path), 0), 1)
		c.address(label, pointerRegister)
		return nil
	}
	if isRuntimeArgumentText(input) {
		return c.emitRuntimeArgumentTextOperand(input, slots, constants, lengthRegister, pointerRegister)
	}
	return fmt.Errorf("文件路径需要静态文本或 argumentText（整数下标）")
}
func (c *windowCode) emitRuntimeFileLoad(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if isModuleRead(input) {
		return c.emitModuleRead(input, constants, slots)
	}
	if !isRuntimeFileRead(input) {
		return fmt.Errorf("文件读取需要 readFile（静态或参数文本路径）")
	}
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 26, 27); err != nil {
		return err
	}
	c.movRegister(0, 27)
	c.movImmediate(1, 0) // O_RDONLY
	c.movImmediate(2, 0)
	c.call(c.fileOpenIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(11) // b.lt: open failed before stdout
	c.movRegister(22, 0)         // read-only descriptor
	return c.emitRuntimeFileContents(false)
}

// Read and close the descriptor in x22, returning its UTF-8 mapping in x21/x28.
func (c *windowCode) emitRuntimeFileContents(trackModule bool) error {
	const maximum = moduleContentSize
	if err := c.emitRegisterImmediate(0, maximum); err != nil {
		return err
	}
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.movRegister(21, 0)
	if trackModule {
		c.words = append(c.words, storeIntegerSlot(21, 31, moduleMappingSlot))
	}
	c.movImmediate(28, 0) // bytes read

	readLoop := len(c.words)
	if err := c.emitRegisterImmediate(0, maximum); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(28, 0))
	c.branchToRuntimeFailure(10) // b.ge: the legal maximum is 16 MiB-1 bytes
	c.movRegister(0, 22)
	c.words = append(c.words, addRegister(1, 21, 28))
	if err := c.emitRegisterImmediate(2, maximum); err != nil {
		return err
	}
	c.words = append(c.words, subtractRegister(2, 2, 28))
	c.call(c.fileReadIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(11) // b.lt
	readDone := c.conditionalPlaceholder(0)
	c.words = append(c.words, addRegister(28, 28, 0), branch(readLoop-len(c.words)-1))

	finish := len(c.words)
	c.patchConditional(readDone, finish)
	c.movRegister(0, 22)
	c.call(c.fileCloseIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1) // b.ne
	if trackModule {
		if err := c.emitRegisterImmediate(1, -1); err != nil {
			return err
		}
		c.words = append(c.words, storeIntegerSlot(1, 31, moduleFileFD))
	}
	c.emitRuntimeUTF8Validation(21, 28)
	c.words = append(c.words, addRegister(24, 21, 28), movz(23, 10), storeByte(23, 24))
	return nil
}
func (c *windowCode) emitRuntimeTextFile(name string, input value, constants map[string]constantValue, slots map[string]uint32, initialize bool) error {
	if err := c.emitRuntimeFileLoad(input, constants, slots); err != nil {
		return err
	}
	buffer, length, capacity := c.textBuffers[name], c.textLengths[name], c.textCapacities[name]
	if !initialize {
		c.words = append(c.words, loadIntegerSlot(0, 19, buffer))
		c.words = append(c.words, loadIntegerSlot(1, 19, capacity))
		c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		c.branchToRuntimeFailure(1) // b.ne
	}
	c.words = append(c.words, storeIntegerSlot(21, 19, buffer), storeIntegerSlot(28, 19, length))
	if err := c.emitRegisterImmediate(0, 16*1024*1024); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, capacity))
	return nil
}
func (c *windowCode) emitRuntimeFileOutput(input value, constants map[string]constantValue, slots map[string]uint32, writeIndex, exitIndex int) error {
	if err := c.emitRuntimeFileLoad(input, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, addRegister(24, 21, 28), movz(23, 10), storeByte(23, 24))
	c.movImmediate(0, 1)
	c.movRegister(1, 21)
	c.words = append(c.words, addImmediate(2, 28, 1))
	c.call(writeIndex)

	c.movRegister(0, 21)
	if err := c.emitRegisterImmediate(1, 16*1024*1024); err != nil {
		return err
	}
	c.call(c.arrayUnmapIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
	return nil
}
func (c *windowCode) emitRuntimeEnvironmentTextOutputRegisters(bufferRegister, lengthRegister uint32, writeIndex, exitIndex int) {
	c.movRegister(26, bufferRegister)
	c.movRegister(27, lengthRegister)
	c.movImmediate(0, 1)
	c.movRegister(1, 26)
	c.movRegister(2, 27)
	c.call(writeIndex)
	c.addData("runtimeEnvironmentNewline", []byte{'\n'}, 1)
	c.address("runtimeEnvironmentNewline", 1)
	c.movImmediate(0, 1)
	c.movImmediate(2, 1)
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
}
func (c *windowCode) emitRuntimeTextOutputRegisters(bufferRegister, lengthRegister uint32, writeIndex, exitIndex int) {
	c.movImmediate(0, 1)
	c.movRegister(1, bufferRegister)
	c.words = append(c.words, addImmediate(2, lengthRegister, 1))
	c.call(writeIndex)
	c.movImmediate(0, 0)
	c.call(exitIndex)
}
func (c *windowCode) emitRuntimeUTF8Validation(bufferRegister, lengthRegister uint32) {
	type localPatch struct {
		patch conditionalPatch
		label string
	}
	labels := map[string]int{}
	patches := []localPatch{}
	mark := func(label string) { labels[label] = len(c.words) }
	branchLocal := func(label string, condition uint32) {
		patches = append(patches, localPatch{patch: c.conditionalPlaceholder(condition), label: label})
	}
	c.movImmediate(22, 0)
	loop := len(c.words)
	c.words = append(c.words, compareRegister(22, lengthRegister))
	branchLocal("valid", 10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	branchLocal("ascii", 11) // b.lt
	c.words = append(c.words, compareImmediate(23, 0xc2))
	c.branchToRuntimeFailure(3) // b.lo; excludes stray continuation bytes and overlong two-byte leads
	c.words = append(c.words, compareImmediate(23, 0xdf))
	branchLocal("two", 13) // b.le
	c.words = append(c.words, compareImmediate(23, 0xef))
	branchLocal("three", 13) // b.le
	c.words = append(c.words, compareImmediate(23, 0xf4))
	branchLocal("four", 13)     // b.le
	c.branchToRuntimeFailure(8) // b.hi; excludes invalid lead bytes above F4
	mark("ascii")
	c.words = append(c.words, addImmediate(22, 22, 1), branch(loop-len(c.words)-1))
	mark("two")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0xbf))
	c.branchToRuntimeFailure(8) // b.hi
	c.words = append(c.words, addImmediate(22, 22, 1), branch(loop-len(c.words)-1))
	mark("three")
	c.movRegister(25, 23)
	c.words = append(c.words, compareImmediate(25, 0xe0))
	branchLocal("three-e0", 0) // b.eq
	c.words = append(c.words, compareImmediate(25, 0xed))
	branchLocal("three-ed", 0) // b.eq
	mark("three-normal")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0xbf))
	c.branchToRuntimeFailure(8) // b.hi
	threeSecondJoin := c.unconditionalPlaceholder()
	mark("three-e0")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0xa0))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0xbf))
	c.branchToRuntimeFailure(8) // b.hi
	threeE0Join := c.unconditionalPlaceholder()
	mark("three-ed")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0x9f))
	c.branchToRuntimeFailure(8) // b.hi
	threeEDJoin := c.unconditionalPlaceholder()
	mark("three-third")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0xbf))
	c.branchToRuntimeFailure(8) // b.hi
	c.words = append(c.words, addImmediate(22, 22, 1), branch(loop-len(c.words)-1))
	mark("four")
	c.movRegister(25, 23)
	c.words = append(c.words, compareImmediate(25, 0xf0))
	branchLocal("four-f0", 0) // b.eq
	c.words = append(c.words, compareImmediate(25, 0xf4))
	branchLocal("four-f4", 0) // b.eq
	mark("four-normal")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0xbf))
	c.branchToRuntimeFailure(8) // b.hi
	fourSecondJoin := c.unconditionalPlaceholder()
	mark("four-f0")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x90))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0xbf))
	c.branchToRuntimeFailure(8) // b.hi
	fourF0Join := c.unconditionalPlaceholder()
	mark("four-f4")
	c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
	c.branchToRuntimeFailure(10) // b.ge
	c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
	c.branchToRuntimeFailure(3) // b.lo
	c.words = append(c.words, compareImmediate(23, 0x8f))
	c.branchToRuntimeFailure(8) // b.hi
	fourF4Join := c.unconditionalPlaceholder()
	mark("four-third")
	for index := 0; index < 2; index++ {
		c.words = append(c.words, addImmediate(22, 22, 1), compareRegister(22, lengthRegister))
		c.branchToRuntimeFailure(10) // b.ge
		c.words = append(c.words, addRegister(24, bufferRegister, 22), loadByte(23, 24), compareImmediate(23, 0x80))
		c.branchToRuntimeFailure(3) // b.lo
		c.words = append(c.words, compareImmediate(23, 0xbf))
		c.branchToRuntimeFailure(8) // b.hi
	}
	c.words = append(c.words, addImmediate(22, 22, 1), branch(loop-len(c.words)-1))
	mark("valid")
	for _, patch := range patches {
		c.patchConditional(patch.patch, labels[patch.label])
	}
	for _, position := range []int{threeSecondJoin, threeE0Join, threeEDJoin} {
		c.words[position] = branch(labels["three-third"] - position)
	}
	for _, position := range []int{fourSecondJoin, fourF0Join, fourF4Join} {
		c.words[position] = branch(labels["four-third"] - position)
	}
}
func (c *windowCode) emitRuntimeInteger(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if input.kind == "比较" || input.kind == "逻辑" || input.kind == "调用" && (input.text == "equals" || input.text == "notEquals" || input.text == "是否失败" || input.text == "映射有") {
		input = value{kind: "调用", text: "ifElse", args: []value{input, {kind: "整数", text: "1"}, {kind: "整数", text: "0"}}}
	}
	switch input.kind {
	case "整数":
		return c.emitIntegerImmediate(input.text)
	case "引用":
		if offset, exists := slots[input.text]; exists {
			c.words = append(c.words, loadIntegerSlot(0, 19, offset))
			return nil
		}
		constant, exists := constants[input.text]
		if !exists || constant.kind != "整数" {
			return fmt.Errorf("控制台整数表达式无法解析常量：%s", input.text)
		}
		return c.emitIntegerImmediate(fmt.Sprint(constant.integer))
	case "表达式":
		if input.left == nil {
			return fmt.Errorf("缺少整数表达式左操作数")
		}
		if input.operator == "negate" {
			if err := c.emitRuntimeInteger(*input.left, constants, slots); err != nil {
				return err
			}
			c.words = append(c.words, 0xeb0003e0) // negs x0, x0
			c.branchToRuntimeFailure(6)           // b.vs on int64 overflow
			return nil
		}
		if input.right == nil {
			return fmt.Errorf("缺少整数表达式右操作数")
		}
		if err := c.emitRuntimeInteger(*input.left, constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, subImmediate(31, 31, 16), 0xf90003e0)
		if err := c.emitRuntimeInteger(*input.right, constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, 0xaa0003e1, 0xf94003e0, addImmediate(31, 31, 16))
		switch input.operator {
		case "+":
			c.words = append(c.words, 0xab010000) // adds x0, x0, x1
			c.branchToRuntimeFailure(6)           // b.vs
		case "-":
			c.words = append(c.words, 0xeb010000) // subs x0, x0, x1
			c.branchToRuntimeFailure(6)           // b.vs
		case "*":
			c.words = append(c.words, 0x9b417c02, 0x9b017c00, movz(4, 63), 0x9ac42803, compareRegister(2, 3))
			c.branchToRuntimeFailure(1) // b.ne when high product is not sign extension
		case "/":
			c.words = append(c.words, compareImmediate(1, 0))
			c.branchToRuntimeFailure(0) // b.eq
			if err := c.emitRegisterImmediate(2, -9223372036854775808); err != nil {
				return err
			}
			c.words = append(c.words, compareRegister(0, 2))
			notMinimum := c.conditionalPlaceholder(1) // b.ne
			if err := c.emitRegisterImmediate(2, -1); err != nil {
				return err
			}
			c.words = append(c.words, compareRegister(1, 2))
			c.branchToRuntimeFailure(0) // b.eq for min / -1
			c.patchConditional(notMinimum, len(c.words))
			c.words = append(c.words, 0x9ac10c00)
		case "%":
			c.words = append(c.words, compareImmediate(1, 0))
			c.branchToRuntimeFailure(0) // b.eq
			c.words = append(c.words, 0x9ac10c02, 0x9b018040)
		case "<<":
			c.words = append(c.words, compareImmediate(1, 0))
			c.branchToRuntimeFailure(11) // b.lt
			c.words = append(c.words, compareImmediate(1, 63))
			c.branchToRuntimeFailure(12) // b.gt
			c.words = append(c.words, 0xaa0003e2, 0x9ac12000, 0x9ac12803, compareRegister(3, 2))
			c.branchToRuntimeFailure(1) // b.ne if left shift overflows int64
		case ">>":
			c.words = append(c.words, compareImmediate(1, 0))
			c.branchToRuntimeFailure(11) // b.lt
			c.words = append(c.words, compareImmediate(1, 63))
			c.branchToRuntimeFailure(12) // b.gt
			c.words = append(c.words, 0x9ac12800)
		default:
			return fmt.Errorf("不支持的原生整数运算符：%s", input.operator)
		}
		return nil
	case "调用":
		if function, exists := c.runtimeFunctions[input.text]; exists {
			if function.runtimeReturnsResult {
				return fmt.Errorf("运行时结果函数不能进入整数表达式")
			}
			return c.emitRuntimeFunctionCall(input, constants, slots)
		}
		if input.text == "ifElse" && len(input.args) == 3 {
			condition := input.args[0]
			if !containsRuntimeArguments(condition) && !containsRuntimeIntegerInput(condition) && !containsRuntimeTextInput(condition) && !containsIntegerSlotReference(condition, slots) && !containsRuntimeTextVariableReference(condition, c.textBuffers) && !containsTextResultValue(condition) && !containsRuntimeTextReturnCall(condition) && !containsResultStateCall(condition) {
				selected, problem := evaluateBoolean(condition, 0, constants)
				if problem != nil {
					return fmt.Errorf("无法计算整数 ifElse 条件：%w", problem)
				}
				branchIndex := 2
				if selected {
					branchIndex = 1
				}
				return c.emitRuntimeInteger(input.args[branchIndex], constants, slots)
			}
			falsePatches, err := c.emitConditionFalseBranches(condition, constants, slots)
			if err != nil {
				return err
			}
			if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
				return err
			}
			end := c.unconditionalPlaceholder()
			falseStart := len(c.words)
			c.patchConditionFalse(falsePatches, falseStart)
			if err := c.emitRuntimeInteger(input.args[2], constants, slots); err != nil {
				return err
			}
			c.words[end] = branch(len(c.words) - end)
			return nil
		}
		if input.text == "readInteger" && len(input.args) == 0 {
			return c.emitReadInteger(c.temporaryBufferOffset)
		}
		if input.text == "textByteLength" || input.text == "textByteAt" {
			return c.emitTextByteOperation(input, constants, slots)
		}
		if input.text == "textLength" && len(input.args) == 1 {
			if input.args[0].kind == "引用" {
				if _, exists := c.textBuffers[input.args[0].text]; exists {
					return c.emitRuntimeTextLength(input.args[0], constants, slots)
				}
			}
			if isRuntimeTextInput(input.args[0]) || isRuntimeArgumentText(input.args[0]) || input.args[0].kind == "调用" && (input.args[0].text == "textConcat" || input.args[0].text == "substring" || input.args[0].text == "arrayGet" || input.args[0].text == "结果值" || input.args[0].text == "readEnvironment" || input.args[0].text == "readLine") || isTextReturnCall(input.args[0]) {
				return c.emitRuntimeTextLength(input.args[0], constants, slots)
			}
			text, problem := evaluateString(input.args[0], 0, constants)
			if problem != nil {
				return fmt.Errorf("无法计算静态文本码位长度：%w", problem)
			}
			return c.emitIntegerImmediate(strconv.Itoa(len([]rune(text))))
		}
		if input.text == "错误码" && len(input.args) == 1 && (input.args[0].kind == "引用" || runtimeResultOperandShape(input.args[0])) {
			if input.args[0].kind == "引用" {
				result, exists := c.resultSlots[input.args[0].text]
				if !exists {
					return fmt.Errorf("无法解析结果变量：%s", input.args[0].text)
				}
				c.words = append(c.words, loadIntegerSlot(0, 19, result.value), loadIntegerSlot(1, 19, result.tag), compareImmediate(1, 0))
			} else if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
				return err
			} else {
				c.movRegister(2, 0)
				c.movRegister(0, 1)
				c.movRegister(1, 2)
				c.words = append(c.words, compareImmediate(1, 0))
			}
			success := c.conditionalPlaceholder(1)
			c.movImmediate(0, 0)
			c.patchConditional(success, len(c.words))
			return nil
		}
		if input.text == "结果值" && len(input.args) == 1 && (input.args[0].kind == "引用" || runtimeResultOperandShape(input.args[0])) {
			var success conditionalPatch
			var result resultSlot
			if input.args[0].kind == "引用" {
				var exists bool
				result, exists = c.resultSlots[input.args[0].text]
				if !exists {
					return fmt.Errorf("无法解析结果变量：%s", input.args[0].text)
				}
				c.words = append(c.words, loadIntegerSlot(2, 19, result.tag), compareImmediate(2, 1))
				success = c.conditionalPlaceholder(1)
				c.words = append(c.words, loadIntegerSlot(0, 19, result.value))
			} else if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
				return err
			} else {
				c.movRegister(0, 1)
				c.words = append(c.words, compareImmediate(0, 1))
				success = c.conditionalPlaceholder(1)
				c.call(c.exitIndex)
				c.patchConditional(success, len(c.words))
				return nil
			}
			c.call(c.exitIndex)
			c.patchConditional(success, len(c.words))
			c.words = append(c.words, loadIntegerSlot(0, 19, result.value))
			return nil
		}
		if isRuntimeArgumentCount(input) {
			c.words = append(c.words, loadIntegerSlot(0, 19, c.argumentCountOffset))
			return nil
		}
		if input.text == "映射长" && len(input.args) == 1 && input.args[0].kind == "引用" {
			mapSlot, exists := c.mapSlots[input.args[0].text]
			if !exists {
				return fmt.Errorf("无法解析映射变量：%s", input.args[0].text)
			}
			c.words = append(c.words, loadIntegerSlot(0, 19, mapSlot.length))
			return nil
		}
		if input.text == "映射取" && len(input.args) == 3 && input.args[0].kind == "引用" {
			return c.emitRuntimeMapGet(input, constants, slots)
		}
		if input.text == "映射有" && len(input.args) == 2 && input.args[0].kind == "引用" {
			return c.emitRuntimeMapHas(input, constants, slots)
		}
		if input.text == "arrayGet" && len(input.args) == 2 {
			arrayName := ""
			variableArray := false
			if input.args[0].kind == "引用" {
				arrayName = input.args[0].text
				_, variableArray = c.arraySlots[arrayName]
			}
			if !variableArray {
				integer, problem := evaluateArrayAccess(input, constants)
				if problem != nil {
					return fmt.Errorf("无法计算整数数组常量索引：%w", problem)
				}
				if integer.kind != "整数" {
					return fmt.Errorf("控制台数组索引结果不是整数")
				}
				return c.emitIntegerImmediate(strconv.FormatInt(integer.integer, 10))
			}
			if input.args[0].kind != "引用" {
				return fmt.Errorf("原生整数数组索引需要变量数组引用")
			}
			array, exists := c.arraySlots[arrayName]
			if _, hasBase := slots[arrayName]; !exists || !hasBase {
				return fmt.Errorf("原生整数数组索引无法解析数组变量：%s", arrayName)
			}
			if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
				return err
			}
			c.emitRuntimeArrayIndexBounds(array.length)
			c.words = append(c.words, loadIntegerSlot(1, 19, array.data), loadIndexedRegister(0, 1, 0))
			return nil
		}
		if isRuntimeFileWrite(input) {
			return c.emitRuntimeFileWrite(input, constants, slots)
		}
		if isRuntimeFileMode(input) {
			return c.emitRuntimeFileMode(input, constants, slots)
		}
		if isRuntimeFileRemove(input) {
			return c.emitRuntimeFileRemove(input, constants, slots)
		}
		if isRuntimeFileRename(input) {
			return c.emitRuntimeFileRename(input, constants, slots)
		}
		if isRuntimeFileReplace(input) {
			return c.emitRuntimeFileReplace(input, constants, slots)
		}
		if isRuntimeDirectoryMutation(input) {
			return c.emitRuntimeDirectoryMutation(input, constants, slots)
		}
		if isRuntimeFileSize(input) {
			return c.emitRuntimeFileSize(input, constants, slots)
		}
		if isRuntimePathExists(input) {
			return c.emitRuntimePathExists(input, constants, slots)
		}
		if isRuntimeDirectoryCheck(input) {
			return c.emitRuntimeDirectoryCheck(input, constants, slots)
		}
		if input.text == "arrayLength" && len(input.args) == 1 {
			if input.args[0].kind == "引用" {
				if array, exists := c.arraySlots[input.args[0].text]; exists {
					c.words = append(c.words, loadIntegerSlot(0, 19, array.length))
					return nil
				}
			}
			length, problem := evaluateArrayLength(input, constants)
			if problem != nil {
				return fmt.Errorf("无法确定原生整数数组长度：%s", problem.code)
			}
			return c.emitIntegerImmediate(strconv.FormatInt(length, 10))
		}
		if input.text != "bitAnd" && input.text != "bitOr" && input.text != "bitXor" || len(input.args) != 2 {
			return fmt.Errorf("不支持的原生整数调用：%s", input.text)
		}
		if err := c.emitRuntimeInteger(input.args[0], constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, subImmediate(31, 31, 16), 0xf90003e0)
		if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, 0xaa0003e1, 0xf94003e0, addImmediate(31, 31, 16))
		switch input.text {
		case "bitAnd":
			c.words = append(c.words, 0x8a010000)
		case "bitOr":
			c.words = append(c.words, 0xaa010000)
		case "bitXor":
			c.words = append(c.words, 0xca010000)
		}
		return nil
	default:
		return fmt.Errorf("不支持的原生整数表达式：%s", input.kind)
	}
}
func (c *windowCode) emitRuntimeMapFind(name string, key value, constants map[string]constantValue, slots map[string]uint32) error {
	mapSlot, exists := c.mapSlots[name]
	if !exists {
		return fmt.Errorf("无法解析映射变量：%s", name)
	}
	c.words = append(c.words, subImmediate(31, 31, 32))
	if err := c.emitRuntimeInteger(key, constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 0), loadIntegerSlot(1, 19, mapSlot.data), compareImmediate(1, 0))
	missingData := c.conditionalPlaceholder(0)
	if err := c.emitRegisterImmediate(2, -7046029254386353131); err != nil {
		return err
	}
	c.words = append(c.words, multiplyRegister(3, 0, 2))
	c.words = append(c.words, loadIntegerSlot(2, 19, mapSlot.capacity), subImmediate(2, 2, 1), andRegister(3, 3, 2), loadIntegerSlot(1, 19, mapSlot.data), addRegisterShifted(1, 1, 3, 3), loadIntegerSlot(3, 1, 0))
	loop := len(c.words)
	c.words = append(c.words, compareImmediate(3, 0))
	missingEntry := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(4, 3, 0), compareRegister(4, 2))
	tombstone := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(5, 31, 0), compareRegister(4, 5))
	found := c.conditionalPlaceholder(0)
	c.patchConditional(tombstone, len(c.words))
	c.words = append(c.words, loadIntegerSlot(3, 3, 16))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(missingData, len(c.words))
	c.patchConditional(missingEntry, len(c.words))
	c.words = append(c.words, movz(3, 0))
	c.patchConditional(found, len(c.words))
	c.words = append(c.words, addImmediate(31, 31, 32))
	return nil
}
func (c *windowCode) emitRuntimeMapGrow(name string) error {
	mapSlot, exists := c.mapSlots[name]
	if !exists {
		return fmt.Errorf("无法扩容映射变量：%s", name)
	}
	c.words = append(c.words, loadIntegerSlot(1, 19, mapSlot.capacity), loadIntegerSlot(2, 19, mapSlot.data), storeIntegerSlot(2, 31, 72), storeIntegerSlot(1, 31, 80))
	if err := c.emitRegisterImmediate(3, 1<<24); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(1, 3))
	c.branchToRuntimeFailure(10) // capacity doubling is bounded to 24 bits
	c.words = append(c.words, storeIntegerSlot(1, 31, 56), storeIntegerSlot(2, 31, 48), logicalShiftLeft(2, 1, 1), storeIntegerSlot(2, 31, 64), logicalShiftLeft(0, 2, 3))
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, mapSlot.data), movz(7, 0))
	outer := len(c.words)
	c.words = append(c.words, loadIntegerSlot(1, 31, 56), compareRegister(7, 1))
	outerDone := c.conditionalPlaceholder(10) // b.ge
	// 旧桶数组是 8 字节指针槽；空槽说明没有待迁移节点。原实现用新表基址和
	// 新容量掩码探测，导致扩容时错误跳过几乎所有旧桶。
	c.words = append(c.words, loadIntegerSlot(1, 31, 80), subImmediate(1, 1, 1), andRegister(1, 7, 1), loadIntegerSlot(0, 31, 72), addRegisterShifted(0, 0, 1, 3), loadIntegerSlot(0, 0, 0), compareImmediate(0, 0))
	emptyOldBucket := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(3, 31, 72), addRegisterShifted(3, 3, 7, 3), loadIntegerSlot(4, 3, 0))
	inner := len(c.words)
	c.words = append(c.words, compareImmediate(4, 0))
	innerDone := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, loadIntegerSlot(5, 4, 16), loadIntegerSlot(0, 4, 0))
	if err := c.emitRegisterImmediate(2, -7046029254386353131); err != nil {
		return err
	}
	c.words = append(c.words, multiplyRegister(0, 0, 2), loadIntegerSlot(1, 31, 64), subImmediate(1, 1, 1), andRegister(0, 0, 1), loadIntegerSlot(3, 19, mapSlot.data), addRegisterShifted(3, 3, 0, 3), loadIntegerSlot(6, 3, 0), storeIntegerSlot(6, 4, 16), storeIntegerSlot(4, 3, 0))
	c.movRegister(4, 5)
	c.words = append(c.words, branch(inner-len(c.words)))
	// 空旧桶和走完碰撞链都必须进入下一桶；不能跳到重散列收束点，否则
	// 第一个空桶会提前结束整个旧表遍历，后续键全部丢失。
	nextBucket := len(c.words)
	c.words = append(c.words, addImmediate(7, 7, 1), branch(outer-len(c.words)))
	c.patchConditional(innerDone, nextBucket)
	c.patchConditional(emptyOldBucket, nextBucket)
	c.patchConditional(outerDone, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 48), loadIntegerSlot(1, 31, 56), logicalShiftLeft(1, 1, 3))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, loadIntegerSlot(0, 31, 64), storeIntegerSlot(0, 19, mapSlot.capacity))
	return nil
}
func (c *windowCode) emitRuntimeMapSet(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 3 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生映射写入需要 映射置（映射，键，值）")
	}
	name := input.args[0].text
	mapSlot, exists := c.mapSlots[name]
	if !exists {
		return fmt.Errorf("原生映射写入无法解析变量：%s", name)
	}
	c.words = append(c.words, subImmediate(31, 31, 96))
	if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 0))
	if err := c.emitRuntimeInteger(input.args[2], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 8), loadIntegerSlot(0, 19, mapSlot.data), compareImmediate(0, 0))
	dataReady := c.conditionalPlaceholder(1)
	if err := c.emitRegisterImmediate(0, 512); err != nil {
		return err
	}
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, mapSlot.data), movz(0, 64), storeIntegerSlot(0, 19, mapSlot.capacity))
	c.patchConditional(dataReady, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 0))
	if err := c.emitRegisterImmediate(1, -7046029254386353131); err != nil {
		return err
	}
	c.words = append(c.words, multiplyRegister(2, 0, 1))
	c.words = append(c.words, loadIntegerSlot(1, 19, mapSlot.capacity), subImmediate(1, 1, 1), andRegister(2, 2, 1), storeIntegerSlot(2, 31, 16), loadIntegerSlot(1, 19, mapSlot.data), addRegisterShifted(1, 1, 2, 3), storeIntegerSlot(1, 31, 24), loadIntegerSlot(3, 1, 0), storeIntegerSlot(3, 31, 32))
	loop := len(c.words)
	c.words = append(c.words, compareImmediate(3, 0))
	insert := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(4, 3, 0), loadIntegerSlot(5, 31, 0), compareRegister(4, 5))
	found := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(3, 3, 16))
	c.words = append(c.words, branch(loop-len(c.words)))
	foundBlock := len(c.words)
	c.words = append(c.words, loadIntegerSlot(1, 31, 8), storeIntegerSlot(1, 3, 8))
	foundDone := c.unconditionalPlaceholder()
	c.patchConditional(insert, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 19, mapSlot.length), addImmediate(0, 0, 1), logicalShiftLeft(1, 0, 2), loadIntegerSlot(2, 19, mapSlot.capacity), addRegister(3, 2, 2), addRegister(2, 3, 2), compareRegister(1, 2))
	capacityReady := c.conditionalPlaceholder(11) // b.lt when the new load stays below three quarters
	if err := c.emitRuntimeMapGrow(name); err != nil {
		return err
	}
	c.patchConditional(capacityReady, len(c.words))
	if err := c.emitRegisterImmediate(0, 24); err != nil {
		return err
	}
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 40), loadIntegerSlot(1, 31, 0), storeIntegerSlot(1, 0, 0), loadIntegerSlot(1, 31, 8), storeIntegerSlot(1, 0, 8), movz(1, 0), storeIntegerSlot(1, 0, 16), storeIntegerSlot(0, 31, 48), loadIntegerSlot(0, 31, 0))
	if err := c.emitRegisterImmediate(1, -7046029254386353131); err != nil {
		return err
	}
	c.words = append(c.words, multiplyRegister(2, 0, 1), loadIntegerSlot(1, 19, mapSlot.capacity), subImmediate(1, 1, 1), andRegister(2, 2, 1), loadIntegerSlot(1, 19, mapSlot.data), addRegisterShifted(1, 1, 2, 3), loadIntegerSlot(3, 1, 0), loadIntegerSlot(0, 31, 48), storeIntegerSlot(3, 0, 16), storeIntegerSlot(0, 1, 0), loadIntegerSlot(1, 19, mapSlot.length), addImmediate(1, 1, 1), storeIntegerSlot(1, 19, mapSlot.length))
	c.patchConditional(found, foundBlock)
	c.words[foundDone] = branch(len(c.words) - foundDone)
	c.words = append(c.words, addImmediate(31, 31, 96))
	return nil
}
func (c *windowCode) emitRuntimeMapDelete(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 2 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生映射删除需要 删映射键（映射，键）")
	}
	mapSlot, exists := c.mapSlots[input.args[0].text]
	if !exists {
		return fmt.Errorf("原生映射删除无法解析变量：%s", input.args[0].text)
	}
	c.words = append(c.words, subImmediate(31, 31, 32))
	if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
		return err
	}
	// 与 emitRuntimeMapFind 相同：x0 保存查找键，x4 保存墓碑键 -1。
	c.words = append(c.words, storeIntegerSlot(0, 31, 0))
	if err := c.emitRegisterImmediate(2, -7046029254386353131); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(1, 19, mapSlot.data), compareImmediate(1, 0))
	missingData := c.conditionalPlaceholder(0)
	if err := c.emitRegisterImmediate(4, -7046029254386353131); err != nil {
		return err
	}
	c.words = append(c.words, multiplyRegister(5, 0, 2), loadIntegerSlot(4, 19, mapSlot.capacity), subImmediate(4, 4, 1), andRegister(5, 5, 4))
	c.words = append(c.words, loadIntegerSlot(3, 19, mapSlot.data), addRegisterShifted(3, 3, 5, 3), loadIntegerSlot(1, 3, 0))
	probe := len(c.words)
	c.words = append(c.words, compareImmediate(1, 0))
	chainEnd := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(6, 1, 0), compareRegister(6, 4))
	tombstone := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(7, 31, 0), compareRegister(6, 7))
	keyMismatch := c.conditionalPlaceholder(1)
	// x3 是前驱链接槽地址：首节点时是桶槽本身，之后是前一节点的 next 字段。
	// 断链必须写回该槽本身；写到 +16 会破坏相邻桶槽或后继节点的键字段。
	c.words = append(c.words, loadIntegerSlot(6, 1, 16), storeIntegerSlot(6, 3, 0))
	c.words = append(c.words, storeIntegerSlot(2, 1, 0))
	c.words = append(c.words, loadIntegerSlot(0, 19, mapSlot.length), subImmediate(0, 0, 1), storeIntegerSlot(0, 19, mapSlot.length))
	done := c.unconditionalPlaceholder()
	c.patchConditional(tombstone, len(c.words))
	c.patchConditional(keyMismatch, len(c.words))
	c.words = append(c.words, addImmediate(3, 1, 16), loadIntegerSlot(1, 1, 16), branch(probe-len(c.words)))
	c.patchConditional(chainEnd, len(c.words))
	c.patchConditional(missingData, len(c.words))
	c.words[done] = branch(len(c.words) - done)
	c.words = append(c.words, addImmediate(31, 31, 32))
	return nil
}
func (c *windowCode) emitRuntimeMapGet(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 3 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生映射读取需要 映射取（映射，键，默认值）")
	}
	c.words = append(c.words, subImmediate(31, 31, 32))
	if err := c.emitRuntimeInteger(input.args[2], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 0))
	if err := c.emitRuntimeMapFind(input.args[0].text, input.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, compareImmediate(3, 0))
	missing := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIntegerSlot(0, 3, 8))
	done := c.unconditionalPlaceholder()
	c.patchConditional(missing, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 0))
	c.words[done] = branch(len(c.words) - done)
	c.words = append(c.words, addImmediate(31, 31, 32))
	return nil
}
func (c *windowCode) emitRuntimeMapHas(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 2 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生映射状态查询需要 映射有（映射，键）")
	}
	if err := c.emitRuntimeMapFind(input.args[0].text, input.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, compareImmediate(3, 0))
	missing := c.conditionalPlaceholder(0)
	c.movImmediate(0, 1)
	done := c.unconditionalPlaceholder()
	c.patchConditional(missing, len(c.words))
	c.movImmediate(0, 0)
	c.words[done] = branch(len(c.words) - done)
	return nil
}
func (c *windowCode) emitRuntimeArrayIndexBounds(length uint32) {
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(11) // b.lt
	c.words = append(c.words, loadIntegerSlot(1, 19, length), compareRegister(0, 1))
	c.branchToRuntimeFailure(10) // b.ge
}
func (c *windowCode) emitRuntimeTextLength(input value, constants map[string]constantValue, slots map[string]uint32) error {
	switch {
	case input.kind == "引用":
		buffer, bufferExists := c.textBuffers[input.text]
		length, lengthExists := c.textLengths[input.text]
		if !bufferExists || !lengthExists {
			return fmt.Errorf("无法解析运行时文本长度变量：%s", input.text)
		}
		c.words = append(c.words, loadIntegerSlot(22, 19, length), loadIntegerSlot(21, 19, buffer))
	case input.kind == "调用" && input.text == "结果值" && len(input.args) == 1 && input.args[0].kind == "引用":
		result, exists := c.resultSlots[input.args[0].text]
		if !exists || !result.text {
			return fmt.Errorf("无法解析文本结果变量：%s", input.args[0].text)
		}
		c.words = append(c.words, loadIntegerSlot(2, 19, result.tag), compareImmediate(2, 1))
		success := c.conditionalPlaceholder(1)
		c.words = append(c.words, loadIntegerSlot(0, 19, result.value))
		c.call(c.exitIndex)
		c.patchConditional(success, len(c.words))
		c.words = append(c.words, loadIntegerSlot(22, 19, result.length), loadIntegerSlot(21, 19, result.pointer))
	case isRuntimeTextInput(input):
		if err := c.emitReadText(c.temporaryBufferOffset, nil, true, 0, 1, c.exitIndex); err != nil {
			return err
		}
	case isRuntimeArgumentText(input):
		if err := c.emitRuntimeArgumentTextOperand(input, slots, constants, 22, 21); err != nil {
			return err
		}
	case input.kind == "调用" && input.text == "textConcat":
		lengthRegister, err := c.emitRuntimeTextConcat(input, c.temporaryBufferOffset, nil, slots, constants)
		if err != nil {
			return err
		}
		c.emitConsoleStackAddress(21, c.temporaryBufferOffset)
		c.movRegister(22, lengthRegister)
	case input.kind == "调用" && input.text == "substring":
		lengthRegister, err := c.emitRuntimeTextSubstring(input, c.temporaryBufferOffset, constants, slots, false)
		if err != nil {
			return err
		}
		c.emitConsoleStackAddress(21, c.temporaryBufferOffset)
		c.movRegister(22, lengthRegister)
	case input.kind == "调用" && input.text == "arrayGet":
		if err := c.emitRuntimeTextOperand(input, slots, constants, 22, 21, c.temporaryBufferOffset+256); err != nil {
			return err
		}
	case isRuntimeEnvironmentRead(input):
		buffer, length, err := c.emitRuntimeEnvironmentRead(input, constants, slots)
		if err != nil {
			return err
		}
		c.movRegister(22, length)
		c.movRegister(21, buffer)
	case input.kind == "调用" && input.text == "结果值" && len(input.args) == 1 && runtimeResultOperandShape(input.args[0]) && runtimeResultPayload(input.args[0], constants) == "运行时文本":
		if isTextReturnCall(input.args[0]) {
			if err := c.emitRuntimeResultRegisters(input.args[0], constants, slots); err != nil {
				return err
			}
		} else if err := c.emitResultValueRegisters(input.args[0], constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(0, 31, 0), storeIntegerSlot(1, 31, 8), storeIntegerSlot(2, 31, 16), storeIntegerSlot(3, 31, 24))
		c.words = append(c.words, compareImmediate(0, 1))
		success := c.conditionalPlaceholder(1)
		c.movRegister(0, 1)
		c.call(c.exitIndex)
		c.patchConditional(success, len(c.words))
		c.words = append(c.words, loadIntegerSlot(21, 31, 16), loadIntegerSlot(22, 31, 24))
		c.words = append(c.words, compareRegister(22, 0))
		noMapping := c.conditionalPlaceholder(1)
		c.movRegister(0, 21)
		c.words = append(c.words, loadIntegerSlot(1, 31, 16))
		c.movImmediate(1, 0)
		c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		c.branchToRuntimeFailure(1)
		c.patchConditional(noMapping, len(c.words))
		c.words = append(c.words, loadIntegerSlot(1, 31, 0), loadIntegerSlot(0, 31, 8), loadIntegerSlot(22, 31, 24), loadIntegerSlot(21, 31, 16), addImmediate(31, 31, 32))
	case isTextReturnCall(input):
		c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(1, 31, 0), storeIntegerSlot(2, 31, 8), storeIntegerSlot(3, 31, 16))
		if err := c.emitRuntimeFunctionCall(input, constants, slots); err != nil {
			return err
		}
		c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(1, 31, 0), storeIntegerSlot(2, 31, 8), storeIntegerSlot(3, 31, 0))
		c.words = append(c.words, loadIntegerSlot(0, 31, 0), loadIntegerSlot(1, 31, 8))
		c.words = append(c.words, compareRegister(0, 1))
		failure := c.conditionalPlaceholder(1)
		c.words = append(c.words, loadIntegerSlot(22, 31, 0), loadIntegerSlot(21, 31, 8))
		c.words = append(c.words, compareRegister(22, 0))
		noMapping := c.conditionalPlaceholder(1)
		c.movRegister(0, 21)
		c.movImmediate(1, 0)
		c.call(c.arrayUnmapIndex)
		c.words = append(c.words, compareImmediate(0, 0))
		c.branchToRuntimeFailure(1)
		c.patchConditional(failure, len(c.words))
		c.patchConditional(noMapping, len(c.words))
		c.words = append(c.words, loadIntegerSlot(1, 31, 0), loadIntegerSlot(0, 31, 8), addImmediate(31, 31, 16))
	default:
		return fmt.Errorf("无法生成运行时文本长度表达式")
	}
	c.words = append(c.words, movz(23, 0), movz(24, 0))
	loop := len(c.words)
	c.words = append(c.words, compareRegister(23, 22))
	done := c.conditionalPlaceholder(0) // b.eq
	c.words = append(c.words, addRegister(25, 21, 23), loadByte(26, 25), compareImmediate(26, 128))
	nonASCII := c.conditionalPlaceholder(2) // b.hs
	c.words = append(c.words, addImmediate(24, 24, 1))
	skipMultibyte := c.unconditionalPlaceholder()
	multibyte := len(c.words)
	c.patchConditional(nonASCII, multibyte)
	c.words = append(c.words, compareImmediate(26, 192))
	continuation := c.conditionalPlaceholder(3) // b.lo
	c.words = append(c.words, addImmediate(24, 24, 1))
	c.patchConditional(continuation, len(c.words))
	advance := len(c.words)
	c.words = append(c.words, addImmediate(23, 23, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.words[skipMultibyte] = branch(advance - skipMultibyte)
	c.patchConditional(done, len(c.words))
	c.movRegister(0, 24)
	return nil
}
func (c *windowCode) emitArrayMapCall(sizeRegister uint32) error {
	c.movRegister(1, sizeRegister)
	c.movImmediate(2, 3)
	c.movImmediate(3, 0x1002)
	if err := c.emitRegisterImmediate(4, -1); err != nil {
		return err
	}
	c.movImmediate(5, 0)
	c.movImmediate(0, 0)
	c.call(c.arrayMapIndex)
	if err := c.emitRegisterImmediate(25, -1); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(0, 25))
	c.branchToRuntimeFailure(0) // b.eq on MAP_FAILED
	return nil
}
func (c *windowCode) emitRuntimeTextInit(pointer, length, capacity uint32) error {
	if err := c.emitRegisterImmediate(0, 64); err != nil {
		return err
	}
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, pointer), movz(0, 0), storeIntegerSlot(0, 19, length))
	if err := c.emitRegisterImmediate(0, 64); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, capacity))
	return nil
}
func (c *windowCode) emitRuntimeTextResultVariableCopy(input value, buffer, length, capacity uint32) error {
	if err := c.emitRuntimeTextOperand(input, nil, nil, 22, 21, c.temporaryBufferOffset+256); err != nil {
		return err
	}
	return c.emitRuntimeTextCopyOperand(buffer, length, capacity)
}
func (c *windowCode) emitRuntimeTextArrayCopy(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if err := c.emitRuntimeTextOperand(input, slots, constants, 22, 21, c.temporaryBufferOffset+256); err != nil {
		return err
	}
	c.words = append(c.words, addImmediate(0, 22, 1))
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.movRegister(23, 0)
	c.words = append(c.words, movz(24, 0))
	loop := len(c.words)
	c.words = append(c.words, compareRegister(24, 22))
	done := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, addRegister(25, 21, 24), loadByte(26, 25), addRegister(25, 23, 24), storeByte(26, 25), addImmediate(24, 24, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(done, len(c.words))
	c.words = append(c.words, addRegister(25, 23, 22))
	c.movImmediate(26, 10)
	c.words = append(c.words, storeByte(26, 25))
	return nil
}
func (c *windowCode) emitRuntimeArrayInit(input value, array arraySlot, constants map[string]constantValue, slots map[string]uint32) error {
	if input.kind != "调用" || input.text != "array" || len(input.args) == 0 {
		return fmt.Errorf("原生整数数组需要非空 array（…）初始化")
	}
	stride := int64(8)
	if array.text {
		stride = 16
	}
	if err := c.emitRegisterImmediate(0, int64(len(input.args))*stride); err != nil {
		return err
	}
	if err := c.emitArrayMapCall(0); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, array.data))
	if err := c.emitRegisterImmediate(0, int64(len(input.args))); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, array.length))
	if err := c.emitRegisterImmediate(0, int64(len(input.args))); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 19, array.capacity))
	for index, element := range input.args {
		if array.text {
			if err := c.emitRuntimeTextArrayCopy(element, constants, slots); err != nil {
				return err
			}
			if err := c.emitRegisterImmediate(2, int64(index*2)); err != nil {
				return err
			}
			c.words = append(c.words, loadIntegerSlot(1, 19, array.data), storeIndexedRegister(23, 1, 2), addImmediate(1, 1, 8), storeIndexedRegister(22, 1, 2))
			continue
		}
		if err := c.emitRuntimeInteger(element, constants, slots); err != nil {
			return err
		}
		if err := c.emitRegisterImmediate(2, int64(index)); err != nil {
			return err
		}
		c.words = append(c.words, loadIntegerSlot(1, 19, array.data), storeIndexedRegister(0, 1, 2))
	}
	return nil
}
func (c *windowCode) emitRuntimeArraySet(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 3 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生整数数组赋值需要 arraySet（数组，索引，值）")
	}
	name := input.args[0].text
	array, exists := c.arraySlots[name]
	if _, hasBase := slots[name]; !exists || !hasBase {
		return fmt.Errorf("原生整数数组赋值无法解析数组变量：%s", name)
	}
	if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
		return err
	}
	c.emitRuntimeArrayIndexBounds(array.length)
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(0, 31, 0))
	if err := c.emitRuntimeInteger(input.args[2], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(2, 31, 0), loadIntegerSlot(1, 19, array.data), storeIndexedRegister(0, 1, 2), addImmediate(31, 31, 16))
	return nil
}
func (c *windowCode) emitRuntimeTextArraySet(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 3 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生文本数组更新需要 arraySet（数组，索引，值）")
	}
	array, exists := c.arraySlots[input.args[0].text]
	if !exists || !array.text {
		return fmt.Errorf("原生文本数组更新无法解析字符串数组")
	}
	if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
		return err
	}
	c.emitRuntimeArrayIndexBounds(array.length)
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(0, 31, 0))
	if err := c.emitRuntimeTextArrayCopy(input.args[2], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(2, 31, 0), logicalShiftLeft(2, 2, 1), loadIntegerSlot(1, 19, array.data), loadIndexedRegister(28, 1, 2), addImmediate(1, 1, 8), loadIndexedRegister(29, 1, 2), storeIntegerSlot(23, 31, 8), storeIntegerSlot(22, 31, 16))
	c.movRegister(0, 28)
	c.words = append(c.words, addImmediate(1, 29, 1))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, loadIntegerSlot(2, 31, 0), logicalShiftLeft(2, 2, 1), loadIntegerSlot(1, 19, array.data), loadIntegerSlot(23, 31, 8), storeIndexedRegister(23, 1, 2), addImmediate(1, 1, 8), loadIntegerSlot(22, 31, 16), storeIndexedRegister(22, 1, 2), addImmediate(31, 31, 32))
	return nil
}
func (c *windowCode) emitRuntimeTextArrayPush(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 2 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生文本数组追加需要 arrayPush（数组，值）")
	}
	array, exists := c.arraySlots[input.args[0].text]
	if !exists || !array.text {
		return fmt.Errorf("原生文本数组追加无法解析字符串数组")
	}
	if err := c.emitRuntimeTextArrayCopy(input.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 64), storeIntegerSlot(23, 31, 0), storeIntegerSlot(22, 31, 56), loadIntegerSlot(1, 19, array.length), storeIntegerSlot(1, 31, 8))
	if err := c.emitRegisterImmediate(4, 9223372036854775807/16); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(1, 4))
	c.branchToRuntimeFailure(10)
	c.words = append(c.words, addImmediate(6, 1, 1), storeIntegerSlot(6, 31, 24), loadIntegerSlot(2, 19, array.capacity), storeIntegerSlot(2, 31, 48), compareRegister(1, 2))
	appendWithoutGrowth := c.conditionalPlaceholder(11)
	c.words = append(c.words, loadIntegerSlot(2, 19, array.capacity))
	if err := c.emitRegisterImmediate(4, 9223372036854775807/32); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(2, 4))
	newCapacityIsLength := c.conditionalPlaceholder(12)
	c.words = append(c.words, logicalShiftLeft(3, 2, 1), storeIntegerSlot(3, 31, 40))
	capacityReady := c.unconditionalPlaceholder()
	c.patchConditional(newCapacityIsLength, len(c.words))
	c.words = append(c.words, loadIntegerSlot(3, 31, 24), storeIntegerSlot(3, 31, 40))
	c.words[capacityReady] = branch(len(c.words) - capacityReady)
	c.words = append(c.words, loadIntegerSlot(4, 19, array.data), storeIntegerSlot(4, 31, 16), loadIntegerSlot(3, 31, 40), logicalShiftLeft(4, 3, 4))
	if err := c.emitArrayMapCall(4); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 32), movz(7, 0))
	loop := len(c.words)
	c.words = append(c.words, loadIntegerSlot(1, 31, 8), compareRegister(7, 1))
	done := c.conditionalPlaceholder(10)
	c.words = append(c.words, addRegisterShifted(5, 7, 7, 0), loadIntegerSlot(1, 31, 16), loadIndexedRegister(2, 1, 5), addImmediate(1, 1, 8), loadIndexedRegister(3, 1, 5), loadIntegerSlot(1, 31, 32), storeIndexedRegister(2, 1, 5), addImmediate(1, 1, 8), storeIndexedRegister(3, 1, 5), addImmediate(7, 7, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(done, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 16), loadIntegerSlot(1, 31, 48), logicalShiftLeft(1, 1, 4))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, loadIntegerSlot(1, 31, 32), storeIntegerSlot(1, 19, array.data), loadIntegerSlot(2, 31, 40), storeIntegerSlot(2, 19, array.capacity))
	c.patchConditional(appendWithoutGrowth, len(c.words))
	c.words = append(c.words, loadIntegerSlot(2, 31, 8), logicalShiftLeft(2, 2, 1), loadIntegerSlot(1, 19, array.data), loadIntegerSlot(3, 31, 0), storeIndexedRegister(3, 1, 2), addImmediate(1, 1, 8), loadIntegerSlot(3, 31, 56), storeIndexedRegister(3, 1, 2), loadIntegerSlot(2, 31, 24), storeIntegerSlot(2, 19, array.length), addImmediate(31, 31, 64))
	return nil
}
func (c *windowCode) emitRuntimeArrayPush(input value, constants map[string]constantValue, slots map[string]uint32) error {
	if len(input.args) != 2 || input.args[0].kind != "引用" {
		return fmt.Errorf("原生整数数组追加需要 arrayPush（数组，值）")
	}
	name := input.args[0].text
	array, exists := c.arraySlots[name]
	if _, hasBase := slots[name]; !exists || !hasBase {
		return fmt.Errorf("原生整数数组追加无法解析数组变量：%s", name)
	}
	if err := c.emitRuntimeInteger(input.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, subImmediate(31, 31, 64), storeIntegerSlot(0, 31, 0), loadIntegerSlot(1, 19, array.length), storeIntegerSlot(1, 31, 8))
	if err := c.emitRegisterImmediate(4, 9223372036854775807/8); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(1, 4))
	c.branchToRuntimeFailure(10) // b.ge before len + 1 or len * 8
	c.words = append(c.words, addImmediate(6, 1, 1), storeIntegerSlot(6, 31, 24), loadIntegerSlot(2, 19, array.capacity), storeIntegerSlot(2, 31, 48), compareRegister(1, 2))
	appendWithoutGrowth := c.conditionalPlaceholder(11) // b.lt when len < capacity
	c.words = append(c.words, loadIntegerSlot(2, 19, array.capacity))
	if err := c.emitRegisterImmediate(4, 9223372036854775807/16); err != nil {
		return err
	}
	c.words = append(c.words, compareRegister(2, 4))
	newCapacityIsLength := c.conditionalPlaceholder(12) // b.gt; doubling would overflow byte size
	c.words = append(c.words, logicalShiftLeft(3, 2, 1), storeIntegerSlot(3, 31, 40))
	capacityReady := c.unconditionalPlaceholder()
	c.patchConditional(newCapacityIsLength, len(c.words))
	c.words = append(c.words, loadIntegerSlot(3, 31, 24), storeIntegerSlot(3, 31, 40))
	c.words[capacityReady] = branch(len(c.words) - capacityReady)
	c.words = append(c.words, loadIntegerSlot(4, 19, array.data), storeIntegerSlot(4, 31, 16), loadIntegerSlot(3, 31, 40), logicalShiftLeft(4, 3, 3))
	if err := c.emitArrayMapCall(4); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 32), movz(7, 0))
	loop := len(c.words)
	c.words = append(c.words, loadIntegerSlot(1, 31, 8), compareRegister(7, 1))
	done := c.conditionalPlaceholder(10) // b.ge
	c.words = append(c.words, loadIntegerSlot(1, 31, 16), loadIndexedRegister(2, 1, 7), loadIntegerSlot(1, 31, 32), storeIndexedRegister(2, 1, 7), addImmediate(7, 7, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(done, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 16), loadIntegerSlot(1, 31, 48), logicalShiftLeft(1, 1, 3))
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareImmediate(0, 0))
	c.branchToRuntimeFailure(1) // b.ne
	c.words = append(c.words, loadIntegerSlot(1, 31, 32), storeIntegerSlot(1, 19, array.data), loadIntegerSlot(2, 31, 40), storeIntegerSlot(2, 19, array.capacity))
	c.patchConditional(appendWithoutGrowth, len(c.words))
	c.words = append(c.words, loadIntegerSlot(1, 19, array.data), loadIntegerSlot(2, 31, 8), loadIntegerSlot(3, 31, 0), storeIndexedRegister(3, 1, 2), loadIntegerSlot(2, 31, 24), storeIntegerSlot(2, 19, array.length), addImmediate(31, 31, 64))
	return nil
}
func (c *windowCode) emitIntegerImmediate(text string) error {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("整数立即数无效：%w", err)
	}
	return c.emitRegisterImmediate(0, value)
}
func loadIntegerSlot(destination, base, offset uint32) uint32 {
	return 0xf9400000 | (offset/8)<<10 | base<<5 | destination
}
func loadIndexedRegister(destination, base, index uint32) uint32 {
	return 0xf8607800 | index<<16 | base<<5 | destination
}
func storeIndexedRegister(source, base, index uint32) uint32 {
	return 0xf8207800 | index<<16 | base<<5 | source
}
func logicalShiftLeft(destination, source, shift uint32) uint32 {
	return 0xd3400000 | (64-shift)<<16 | (63-shift)<<10 | source<<5 | destination
}
func storeIntegerSlot(source, base, offset uint32) uint32 {
	return 0xf9000000 | (offset/8)<<10 | base<<5 | source
}
func (c *windowCode) emitRegisterImmediate(register uint32, value int64) error {
	bits := uint64(value)
	for shift := uint(0); shift < 64; shift += 16 {
		part := uint16(bits >> shift)
		if shift == 0 {
			c.words = append(c.words, movz(register, part))
		} else {
			c.words = append(c.words, movk(register, part, shift))
		}
	}
	return nil
}
func subImmediate(destination, source, immediate uint32) uint32 {
	return 0xd1000000 | immediate<<10 | source<<5 | destination
}
func (c *windowCode) emitConsoleStackAddress(destination, offset uint32) {
	source := uint32(19)
	if offset >= 4096 {
		c.words = append(c.words, addImmediate(destination, source, offset>>12)|1<<22)
		source = destination
	}
	c.words = append(c.words, addImmediate(destination, source, offset&4095))
}
func addImmediate(destination, source, immediate uint32) uint32 {
	return 0x91000000 | immediate<<10 | source<<5 | destination
}
func addWordImmediate(destination, source, immediate uint32) uint32 {
	return 0x11000000 | immediate<<10 | source<<5 | destination
}
func branch(displacement int) uint32 {
	return 0x14000000 | uint32(displacement)&0x03ffffff
}
func branchLink(displacement int) uint32 {
	return 0x94000000 | uint32(displacement)&0x03ffffff
}
func branchCondition(displacement int, condition uint32) uint32 {
	return 0x54000000 | (uint32(displacement)&0x7ffff)<<5 | condition
}
func loadByte(destination, base uint32) uint32 {
	return 0x39400000 | base<<5 | destination
}
// ldrb wRt, [Xn, Xm]
func loadByteIndex(destination, base, offset uint32) uint32 {
	return 0x38606800 | base<<16 | offset<<13 | 2<<5 | destination
}
func storeByte(source, base uint32) uint32 {
	return 0x39000000 | base<<5 | source
}
func compareImmediate(register, immediate uint32) uint32 {
	return 0xf100001f | immediate<<10 | register<<5
}
func compareWordImmediate(register, immediate uint32) uint32 {
	return 0x7100001f | immediate<<10 | register<<5
}
func subtractWordImmediate(destination, source, immediate uint32) uint32 {
	return 0x51000000 | immediate<<10 | source<<5 | destination
}
func addRegister(destination, left, right uint32) uint32 {
	return 0x8b000000 | right<<16 | left<<5 | destination
}
func addRegisterShifted(destination, left, right, shift uint32) uint32 {
	return 0x8b000000 | right<<16 | shift<<10 | left<<5 | destination
}
func subtractRegister(destination, left, right uint32) uint32 {
	return 0xcb000000 | right<<16 | left<<5 | destination
}
func multiplyRegister(destination, left, right uint32) uint32 {
	return 0x9b007c00 | right<<16 | left<<5 | destination
}
func andRegister(destination, left, right uint32) uint32 {
	return 0x8a000000 | right<<16 | left<<5 | destination
}
func compareRegister(left, right uint32) uint32 {
	return 0xeb00001f | right<<16 | left<<5
}
func movk(register uint32, immediate uint16, shift uint) uint32 {
	return 0xf2800000 | uint32(shift/16)<<21 | uint32(immediate)<<5 | register
}
