// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import "strconv"

func textByteIssue(pos sourcePosition, actual string) *issue {
	return &issue{code: "S-3072", expected: "textByteLength（文本）或 textByteAt（文本，范围内的字节下标）", actual: actual, suggestion: "将运行时文本存入变量，并使用从 0 开始的整数下标", pos: pos}
}

func textByteType(call value, environment map[string]constantValue) (string, *issue) {
	indexed := call.text == "textByteAt"
	if indexed && len(call.args) != 2 || !indexed && len(call.args) != 1 {
		return "", textByteIssue(call.pos, strconv.Itoa(len(call.args))+" 个参数")
	}
	operand := call.args[0]
	kind, problem := constantType(operand, environment)
	if problem != nil {
		return "", problem
	}
	// An evaluated index must not overwrite its source in a shared scratch buffer.
	if kind != "字符串" && !(kind == "运行时字符串" && (operand.kind == "引用" ||
		isRuntimeArgumentText(operand) || isTextResultValue(operand, environment) ||
		operand.kind == "调用" && operand.text == "arrayGet" && len(operand.args) == 2 && operand.args[0].kind == "引用")) {
		return "", textByteIssue(operand.pos, kind+"（缺少稳定的文本存储）")
	}
	if !indexed && kind == "字符串" {
		return "整数", nil
	}
	if indexed {
		subscript := call.args[1]
		indexKind, problem := constantType(subscript, environment)
		if problem != nil {
			return "", problem
		}
		if indexKind != "整数" && indexKind != "运行时整数" {
			return "", textByteIssue(subscript.pos, indexKind+"下标")
		}
		if indexKind == "整数" {
			index, problem := integerValue(subscript, environment)
			if problem != nil {
				return "", problem
			}
			if index < 0 {
				return "", textByteIssue(subscript.pos, strconv.FormatInt(index, 10))
			}
			if kind == "字符串" {
				text, problem := evaluateString(operand, 0, environment)
				if problem != nil {
					return "", problem
				}
				if index >= int64(len(text)) {
					return "", textByteIssue(subscript.pos, strconv.FormatInt(index, 10))
				}
				return "整数", nil
			}
		}
	}
	return "运行时整数", nil
}

func staticTextByteValue(call value, environment map[string]constantValue) (int64, *issue) {
	indexed := call.text == "textByteAt"
	if indexed && len(call.args) != 2 || !indexed && len(call.args) != 1 {
		return 0, textByteIssue(call.pos, strconv.Itoa(len(call.args))+" 个参数")
	}
	text, problem := evaluateString(call.args[0], 0, environment)
	if problem != nil {
		return 0, problem
	}
	if !indexed {
		return int64(len(text)), nil
	}
	index, problem := integerValue(call.args[1], environment)
	if problem != nil {
		return 0, problem
	}
	if index < 0 || index >= int64(len(text)) {
		return 0, textByteIssue(call.args[1].pos, strconv.FormatInt(index, 10))
	}
	return int64(text[index]), nil
}

func (c *windowCode) emitTextByteOperation(call value, constants map[string]constantValue, slots map[string]uint32) error {
	operand := call.args[0]
	if buffer, stored := c.textBuffers[operand.text]; operand.kind == "引用" && stored {
		c.words = append(c.words, loadIntegerSlot(21, 19, buffer), loadIntegerSlot(22, 19, c.textLengths[operand.text]))
	} else if text, problem := evaluateString(operand, 0, constants); problem == nil {
		if call.text == "textByteLength" {
			return c.emitRegisterImmediate(0, int64(len(text)))
		}
		if err := c.emitRegisterImmediate(22, int64(len(text))); err != nil {
			return err
		}
		label := "textBytes" + strconv.Itoa(len(c.dataLabels))
		c.addData(label, []byte(text), 1)
		c.address(label, 21)
	} else if err := c.emitRuntimeTextOperand(operand, slots, constants, 22, 21, c.temporaryBufferOffset); err != nil {
		return err
	}
	if call.text == "textByteLength" {
		c.movRegister(0, 22)
		return nil
	}
	c.words = append(c.words, subImmediate(31, 31, 16), storeIntegerSlot(21, 31, 0), storeIntegerSlot(22, 31, 8))
	if err := c.emitRuntimeInteger(call.args[1], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, loadIntegerSlot(1, 31, 0), loadIntegerSlot(2, 31, 8), addImmediate(31, 31, 16), compareRegister(0, 2))
	c.branchToRuntimeFailure(2) // b.hs rejects negative indices as well as indices at or above the byte length.
	c.words = append(c.words, addRegister(1, 1, 0), loadByte(0, 1))
	return nil
}
