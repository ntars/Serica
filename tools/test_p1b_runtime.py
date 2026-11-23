# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Execute native ARM64 integer and bounded text programs emitted by stage0."""

import pathlib
import re
import resource
import signal
import subprocess
import sys
import tempfile

from native_artifact import verify_native_console_artifact


ROOT = pathlib.Path(__file__).resolve().parent.parent
SERICA = ROOT / "build" / "stage0" / "serica"
MAX_RUNTIME_TEXT_CAPACITY = 16 * 1024 * 1024
VARIABLE_FIXTURE = (ROOT / "examples" / "p1b-runtime-variable.serica").read_text(encoding="utf-8")
VARIABLE_ARITHMETIC_FIXTURE = (ROOT / "examples" / "p1b-runtime-variable-arithmetic.serica").read_text(encoding="utf-8")
INTEGER_EXTENDED_FIXTURE = (ROOT / "examples" / "p1b-runtime-integer-extended.serica").read_text(encoding="utf-8")
RUNTIME_ARITHMETIC_FIXTURE = (ROOT / "examples" / "p1b-runtime-checked-arithmetic.serica").read_text(encoding="utf-8")
INTEGER_SELECTION_FIXTURE = (ROOT / "examples" / "p1b-runtime-integer-selection.serica").read_text(encoding="utf-8")
INTEGER_COMPARISON_FIXTURE = (ROOT / "examples" / "p1b-runtime-integer-comparison.serica").read_text(encoding="utf-8")
INTEGER_CONTROL_FLOW_FIXTURE = (ROOT / "examples" / "p1b-runtime-integer-control-flow.serica").read_text(encoding="utf-8")
LOOP_CONTROL_FIXTURE = (ROOT / "examples" / "p1b-runtime-loop-control.serica").read_text(encoding="utf-8")
INTEGER_IF_FIXTURE = (ROOT / "examples" / "p1b-runtime-integer-if.serica").read_text(encoding="utf-8")
LOGICAL_CONDITION_FIXTURE = (ROOT / "examples" / "p1b-runtime-logical-conditions.serica").read_text(encoding="utf-8")
INTEGER_ARRAY_FIXTURE = (ROOT / "examples" / "p1b-runtime-integer-array.serica").read_text(encoding="utf-8")
DYNAMIC_ARRAY_FIXTURE = (ROOT / "examples" / "p1b-runtime-dynamic-array.serica").read_text(encoding="utf-8")
RUNTIME_TEXT_VARIABLE_FIXTURE = (ROOT / "examples" / "p1b-runtime-text-variable.serica").read_text(encoding="utf-8")
RUNTIME_BOOLEAN_FIXTURE = (ROOT / "examples" / "p1b-runtime-boolean-variable.serica").read_text(encoding="utf-8")
RUNTIME_BOOLEAN_ARRAY_FIXTURE = (ROOT / "examples" / "p1b-runtime-boolean-array.serica").read_text(encoding="utf-8")
RUNTIME_BOOLEAN_EQUALITY_FIXTURE = (ROOT / "examples" / "p1b-runtime-boolean-equality.serica").read_text(encoding="utf-8")
RUNTIME_BOOLEAN_OUTPUT_FIXTURE = (ROOT / "examples" / "p1b-runtime-boolean-output.serica").read_text(encoding="utf-8")
RUNTIME_RESULT_FIXTURE = (ROOT / "examples" / "p1b-runtime-result.serica").read_text(encoding="utf-8")
RUNTIME_HASH_MAP_FIXTURE = (ROOT / "examples" / "p1b-runtime-hash-map.serica").read_text(encoding="utf-8")
RECURSIVE_FUNCTION_FIXTURE = (ROOT / "examples" / "p1b-runtime-recursive-function.serica").read_text(encoding="utf-8")
MAP_GROWTH_SOURCE = (
    "变量 表=映射（）。\n变量 索引=0。\n"
    "凡（索引<200）【赋值 表=映射置（表，索引，索引+1000）。赋值 索引=索引+1。】。\n"
    "程序=【能力=array（『stdout』，『map.write』），"
    "输出=映射长（表）*100000000+映射取（表，0，0）*10000+"
    "映射取（表，100，0）*100+映射取（表，199，0）】。\n"
)
STATIC_BOOLEAN_EQUALITY_FIXTURE = (ROOT / "examples" / "p1b-static-boolean-equality.serica").read_text(encoding="utf-8")


def source_declares_runtime_text_variable(source_text: str) -> bool:
    """Return whether a source declares a text variable that needs an mmap buffer."""
    text_initializers = ("=readText（", "=textConcat（", "=substring（", "=结果值（")
    return any(
        line.lstrip().startswith("变量 ") and any(marker in line for marker in text_initializers)
        for line in source_text.splitlines()
    )
CASES = [
    ("6*7", "程序=【输出=6*7】。\n", "42\n"),
    ("（1+2）*3", "程序=【输出=（1+2）*3】。\n", "9\n"),
    ("-7/2", "程序=【输出=-7/2】。\n", "-3\n"),
    ("bitXor", "程序=【输出=bitXor（-1，2）】。\n", "-3\n"),
    (
        "int64 minimum",
        "程序=【输出=-9223372036854775808】。\n",
        "-9223372036854775808\n",
    ),
    (
        "bitAnd",
        "程序=【输出=bitAnd（9223372036854775807，255）】。\n",
        "255\n",
    ),
    ("bitOr", "程序=【输出=bitOr（128，3）】。\n", "131\n"),
    ("10-15", "程序=【输出=10-15】。\n", "-5\n"),
    ("constant reference", "常量 基数=40。\n程序=【输出=基数+2】。\n", "42\n"),
    ("static boolean and text inequality", STATIC_BOOLEAN_EQUALITY_FIXTURE, "11\n"),
    (
        "pure nested function compile-time evaluation",
        "函数 倍增（数值）=数值*2。\n"
        "函数 三倍（数值）=倍增（数值）+数值。\n"
        "程序=【能力=array（『stdout』），输出=三倍（14）】。\n",
        "42\n",
    ),
    (
        "variable initialization and assignment",
        VARIABLE_FIXTURE,
        "42\n",
    ),
    (
        "variable copy before reassignment",
        "变量 原值=40。\n变量 副本=原值。\n赋值 原值=7。\n程序=【输出=副本】。\n",
        "40\n",
    ),
    (
        "negative variable",
        "变量 数值=-5。\n赋值 数值=6。\n程序=【输出=数值】。\n",
        "6\n",
    ),
    (
        "variable addition and multiplication",
        VARIABLE_ARITHMETIC_FIXTURE,
        "42\n",
    ),
    (
        "variable subtraction and division",
        "变量 数值=10。\n赋值 数值=数值-16。\n变量 分母=2。\n赋值 数值=数值/分母。\n程序=【输出=数值】。\n",
        "-3\n",
    ),
    (
        "variable bitwise",
        "变量 左值=-1。\n变量 右值=2。\n赋值 左值=bitXor（左值，右值）。\n程序=【输出=左值】。\n",
        "-3\n",
    ),
    (
        "negative remainder",
        "程序=【输出=-17%5】。\n",
        "-2\n",
    ),
    (
        "minimum remainder",
        "程序=【输出=-9223372036854775808%-1】。\n",
        "0\n",
    ),
    (
        "shift precedence",
        "程序=【输出=1+2<<3】。\n",
        "24\n",
    ),
    (
        "arithmetic right shift",
        "程序=【输出=-8>>2】。\n",
        "-2\n",
    ),
    (
        "negative left shift",
        "程序=【输出=-3<<2】。\n",
        "-12\n",
    ),
    (
        "variable remainder and shift",
        INTEGER_EXTENDED_FIXTURE,
        "-4\n",
    ),
    (
        "static integer selection",
        "程序=【输出=ifElse（equals（6，6），40+2，-1）】。\n",
        "42\n",
    ),
    ("static less than", "程序=【输出=ifElse（1<2，42，-1）】。\n", "42\n"),
    ("static less equal", "程序=【输出=ifElse（2<=2，42，-1）】。\n", "42\n"),
    ("static greater than", "程序=【输出=ifElse（3>2，42，-1）】。\n", "42\n"),
    ("static greater equal", "程序=【输出=ifElse（3>=3，42，-1）】。\n", "42\n"),
    ("static equal operator", "程序=【输出=ifElse（3==3，42，-1）】。\n", "42\n"),
    ("static not equal", "程序=【输出=ifElse（notEquals（3，2），42，-1）】。\n", "42\n"),
    ("static boolean true output", "程序=【输出=1<2】。\n", "1\n"),
    ("static boolean false output", "程序=【输出=1>2】。\n", "0\n"),
    ("static logical boolean output", "程序=【输出=1<2 且 2<3】。\n", "1\n"),
    ("static logical and", "程序=【输出=ifElse（1<2 且 2<=2，42，-1）】。\n", "42\n"),
    ("static logical or precedence", "程序=【输出=ifElse（1<2 故 2<1 且 3<2，42，-1）】。\n", "42\n"),
    ("nested condition and loop statements", INTEGER_CONTROL_FLOW_FIXTURE, "8\n"),
    (
        "nested loop break and continue",
        LOOP_CONTROL_FIXTURE,
        "112233\n",
    ),
    (
        "constant true loop exits with break",
        "变量 次数=0。凡（1==1）【赋值 次数=次数+1。"
        "若（次数==3）【跳出。】。】。程序=【输出=次数】。\n",
        "3\n",
    ),
    ("dynamic integer array length", "变量 数组=array（10，20，30）。\n程序=【输出=arrayLength（数组）】。\n", "3\n"),
    ("constant array indexed value", "常量 数组=array（10，20，30）。\n程序=【输出=arrayGet（数组，1）】。\n", "20\n"),
    ("inline array literal indexed value", "程序=【输出=arrayGet（array（10，20，30），2）】。\n", "30\n"),
]
DYNAMIC_CASES = [
    ("runtime boolean variable true", RUNTIME_BOOLEAN_FIXTURE, "1\n5\n", "42\n", "b.eq", True),
    ("runtime boolean variable false", RUNTIME_BOOLEAN_FIXTURE, "0\n0\n", "-1\n", "b.eq", True),
    ("runtime boolean comparison output true", "程序=【输出=readInteger（）<5】。\n", "4\n", "1\n", "b.ge", False),
    ("runtime boolean comparison output false", "程序=【输出=readInteger（）<5】。\n", "5\n", "0\n", "b.ge", False),
    ("runtime boolean variable output", "变量 标志=readInteger（）>0。程序=【输出=标志】。\n", "1\n", "1\n", "b.le", True),
    ("runtime boolean array output", RUNTIME_BOOLEAN_OUTPUT_FIXTURE, "1\n1\n", "0\n", (), False),
    ("runtime result success value and error code", "变量 结果=成功（readInteger（））。\n程序=【输出=结果值（结果）*100+错误码（结果）】。\n", "42\n", "4200\n", (), True),
    ("runtime result failure code", "变量 结果=失败（readInteger（））。\n程序=【输出=错误码（结果）】。\n", "7\n", "7\n", (), True),
    ("runtime result failure status", "变量 结果=失败（readInteger（））。\n程序=【输出=是否失败（结果）】。\n", "7\n", "1\n", (), True),
    ("runtime result success status", "变量 结果=成功（readInteger（））。\n程序=【输出=是否失败（结果）】。\n", "7\n", "0\n", (), True),
    ("runtime result copy and failure code", RUNTIME_RESULT_FIXTURE, "42\n9\n", "4209\n", (), True),
    ("runtime result reassignment observed", "变量 结果=成功（readInteger（））。\n赋值 结果=失败（readInteger（））。\n程序=【输出=错误码（结果）*100+ifElse（是否失败（结果），1，0）】。\n", "42\n9\n", "901\n", (), True),
    ("runtime addition", "程序=【输出=readInteger（）+5】。\n", "37\n", "42\n", "adds\tx0, x0, x1", False),
    ("runtime subtraction", "程序=【输出=readInteger（）-5】。\n", "47\n", "42\n", "subs\tx0, x0, x1", False),
    ("runtime multiplication", "程序=【输出=readInteger（）*7】。\n", "6\n", "42\n", "smulh\tx2, x0, x1", False),
    ("runtime division", "程序=【输出=readInteger（）/2】。\n", "-84\n", "-42\n", "sdiv\tx0, x0, x1", False),
    ("runtime remainder", "程序=【输出=readInteger（）%5】。\n", "-17\n", "-2\n", "msub\tx0, x2, x1, x0", False),
    ("runtime minimum remainder", "程序=【输出=readInteger（）%readInteger（）】。\n", "-9223372036854775808\n-1\n", "0\n", "msub\tx0, x2, x1, x0", False),
    ("runtime left shift", "程序=【输出=readInteger（）<<3】。\n", "5\n", "40\n", "lsl\tx0, x0, x1", False),
    ("runtime right shift", "程序=【输出=readInteger（）>>2】。\n", "-168\n", "-42\n", "asr\tx0, x0, x1", False),
    ("runtime bitAnd", "程序=【输出=bitAnd（readInteger（），255）】。\n", "511\n", "255\n", "and\tx0, x0, x1", False),
    ("runtime bitOr", "程序=【输出=bitOr（readInteger（），3）】。\n", "128\n", "131\n", "orr\tx0, x0, x1", False),
    ("runtime bitXor", "程序=【输出=bitXor（readInteger（），43）】。\n", "-1\n", "-44\n", "eor\tx0, x0, x1", False),
    ("runtime negation", "程序=【输出=-readInteger（）】。\n", "42\n", "-42\n", "negs\tx0, x0", False),
    (
        "runtime variable arithmetic",
        RUNTIME_ARITHMETIC_FIXTURE,
        "37\n",
        "42\n",
        "adds\tx0, x0, x1",
        True,
    ),
    (
        "runtime integer selection true",
        "程序=【输出=ifElse（equals（readInteger（），5），readInteger（）+1，-1）】。\n",
        "5\n36\n",
        "37\n",
        "b.ne",
        False,
    ),
    (
        "runtime variable selection false",
        INTEGER_SELECTION_FIXTURE,
        "4\n",
        "-1\n",
        "b.ne",
        True,
    ),
    ("runtime less than", "程序=【输出=ifElse（readInteger（）<5，42，-1）】。\n", "4\n", "42\n", "b.ge", False),
    ("runtime less equal", "程序=【输出=ifElse（readInteger（）<=5，42，-1）】。\n", "5\n", "42\n", "b.gt", False),
    ("runtime greater than", "程序=【输出=ifElse（readInteger（）>5，42，-1）】。\n", "6\n", "42\n", "b.le", False),
    ("runtime greater equal", INTEGER_COMPARISON_FIXTURE, "10\n", "1\n", "b.lt", True),
    ("runtime greater equal false", INTEGER_COMPARISON_FIXTURE, "9\n", "0\n", "b.lt", True),
    ("runtime equal operator", "程序=【输出=ifElse（readInteger（）==5，42，-1）】。\n", "5\n", "42\n", "b.ne", False),
    ("runtime not equal", "程序=【输出=ifElse（notEquals（readInteger（），5），42，-1）】。\n", "4\n", "42\n", "b.eq", False),
    ("runtime if true branch", INTEGER_IF_FIXTURE, "10\n", "11\n", "b.lt", True),
    ("runtime if false branch", INTEGER_IF_FIXTURE, "9\n", "8\n", "b.lt", True),
    ("logical and short circuit", LOGICAL_CONDITION_FIXTURE, "1\n", "11\n", "b.ge", True),
    ("logical and/or true path", LOGICAL_CONDITION_FIXTURE, "-1\n7\n", "9\n", "b.eq", True),
    ("logical composition false path", LOGICAL_CONDITION_FIXTURE, "-1\n2\n", "0\n", "b.lt", True),
    (
        "runtime fixed integer array update and read",
        INTEGER_ARRAY_FIXTURE,
        "5\n1\n42\n",
        "42\n",
        ("str\tx0, [x19]", "str\tx2, [x1]", "ldr\tx0, [x1]", "b.ge"),
        False,
    ),
    ("dynamic array grows and reads appended value", "变量 数组=array（5）。\n赋值 数组=arrayPush（数组，readInteger（））。\n赋值 数组=arrayPush（数组，readInteger（））。\n程序=【输出=arrayGet（数组，arrayLength（数组）-1）】。\n", "40\n42\n", "42\n", (), False),
    ("array push evaluates aliased value before growth", "变量 数组=array（readInteger（））。\n赋值 数组=arrayPush（数组，arrayGet（数组，0））。\n程序=【输出=arrayGet（数组，1）】。\n", "5\n", "5\n", (), False),
    ("dynamic array loop growth and tail update", DYNAMIC_ARRAY_FIXTURE, "0\n", "99\n", (), False),
    ("dynamic array retains contents across geometric growth", "变量 数组=array（7）。\n变量 索引=readInteger（）。\n凡（索引<20）【赋值 数组=arrayPush（数组，索引）。赋值 索引=索引+1。】。\n程序=【输出=arrayGet（数组，0）*100+arrayGet（数组，20）】。\n", "0\n", "719\n", (), False),
    ("array set accepts index after growth", "变量 数组=array（10）。\n赋值 数组=arrayPush（数组，readInteger（））。\n赋值 数组=arraySet（数组，1，99）。\n程序=【输出=arrayGet（数组，1）】。\n", "42\n", "99\n", (), False),
    ("runtime boolean array true with length", RUNTIME_BOOLEAN_ARRAY_FIXTURE, "1\n1\n0\n2\n", "4203\n", (), False),
    ("runtime boolean array false with length", RUNTIME_BOOLEAN_ARRAY_FIXTURE, "0\n1\n1\n0\n0\n", "-97\n", (), False),
    ("runtime boolean array equality true", RUNTIME_BOOLEAN_EQUALITY_FIXTURE, "1\n1\n0\n", "42\n", (), False),
    ("runtime boolean array equality false", RUNTIME_BOOLEAN_EQUALITY_FIXTURE, "1\n0\n0\n", "-1\n", (), False),
    ("runtime boolean array inequality true", RUNTIME_BOOLEAN_EQUALITY_FIXTURE.replace("equals（", "notEquals（"), "1\n0\n0\n", "42\n", (), False),
    ("runtime boolean array inequality false", RUNTIME_BOOLEAN_EQUALITY_FIXTURE.replace("equals（", "notEquals（"), "1\n1\n0\n", "-1\n", (), False),
    (
        "effectful builtin as a single-use function argument",
        "函数 加一（数值）能力=array（『stdin』）=数值+1。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=加一（readInteger（））】。\n",
        "41\n",
        "42\n",
        (),
        False,
    ),
    (
        "effectful user function as a single-use function argument",
        "函数 读取（）能力=array（『stdin』）=readInteger（）。\n"
        "函数 加一（数值）能力=array（『stdin』）=数值+1。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=加一（读取（））】。\n",
        "41\n",
        "42\n",
        (),
        False,
    ),
]
MAP_CASES = [
    ("collision update and dynamic key", RUNTIME_HASH_MAP_FIXTURE, "1\n7\n8\n14\n", "2001408\n"),
    ("twenty-one entries behind sixty-four buckets", "变量 表=映射（）。\n变量 索引=0。\n凡（索引<20）【赋值 表=映射置（表，索引，索引*索引）。赋值 索引=索引+1。】。\n赋值 表=映射置（表，65，777）。\n程序=【能力=array（『stdout』，『map.write』），输出=映射长（表）*1000000+映射取（表，14，0）*100+映射取（表，99，-1）】。\n", "", "21019599\n"),
    ("two hundred entries grow and rehash", MAP_GROWTH_SOURCE, "", "20010111199\n"),
    ("overwrite after growth preserves length", MAP_GROWTH_SOURCE.replace("赋值 索引=索引+1。】。", "赋值 索引=索引+1。】。\n赋值 表=映射置（表，100，-7）。"), "", "20010000499\n"),
    (
        "function map write propagation",
        "函数 写入（表）能力=array（『map.write』）=映射置（表，1，7）。\n"
        "函数 代理（表）能力=array（『map.write』）=写入（表）。\n"
        "变量 表=映射（）。赋值 表=代理（表）。\n"
        "程序=【能力=array（『stdout』，『map.write』），"
        "输出=映射长（表）*100+映射取（表，1，0）】。\n",
        "",
        "107\n",
    ),
    ("integer endpoints and defaults", "变量 表=映射（）。\n赋值 表=映射置（表，-9223372036854775808，42）。\n赋值 表=映射置（表，9223372036854775807，-7）。\n程序=【能力=array（『stdout』，『map.write』），输出=映射取（表，-9223372036854775808，0）*100+映射取（表，9223372036854775807，0）+映射取（表，0，7）】。\n", "", "4200\n"),
    ("membership true", "变量 表=映射（）。\n赋值 表=映射置（表，readInteger（），readInteger（））。\n程序=【能力=array（『stdout』，『stdin』，『map.write』），输出=映射有（表，5）】。\n", "5\n42\n", "1\n"),
    ("membership false", "变量 表=映射（）。\n赋值 表=映射置（表，readInteger（），readInteger（））。\n程序=【能力=array（『stdout』，『stdin』，『map.write』），输出=映射有（表，5）】。\n", "6\n42\n", "0\n"),
]
RESULT_FAILURE_CASES = [
    ("static failed result value", "变量 结果=失败（7）。\n程序=【输出=结果值（结果）】。\n", "", 7),
    ("dynamic failed result value", "变量 结果=失败（readInteger（））。\n程序=【输出=结果值（结果）】。\n", "23\n", 23),
    ("dynamic error zero", "变量 结果=失败（readInteger（））。\n程序=【输出=结果值（结果）】。\n", "0\n", 1),
    ("dynamic error above byte", "变量 结果=失败（readInteger（））。\n程序=【输出=结果值（结果）】。\n", "256\n", 1),
    ("failed result inside arithmetic", "变量 结果=失败（7）。\n程序=【输出=结果值（结果）+1】。\n", "", 7),
    (
        "text result replaced by failure",
        "变量 结果=成功（『不应输出』）。\n赋值 结果=失败（9）。\n程序=【输出=结果值（结果）】。\n",
        "",
        9,
    ),
]
RESULT_FUNCTION_CASES = [
    (
        "integer function success boundary",
        "函数 校验（数值）=ifElse（数值<0，失败（31），成功（数值*2））。\n"
        "变量 输入=readInteger（）。\n变量 结果=校验（输入）。\n变量 显示=0。\n"
        "若（是否失败（结果））【赋值 显示=错误码（结果）。】否【赋值 显示=结果值（结果）。】。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=显示】。\n",
        "21\n",
        "42\n",
        0,
    ),
    (
        "integer function failure boundary",
        (ROOT / "examples" / "p1b-runtime-result-function.serica").read_text(encoding="utf-8"),
        "-1\n",
        "31\n",
        0,
    ),
    (
        "nested function result copy observes success",
        "函数 内核（数值）=ifElse（数值==0，失败（17），成功（数值+1））。\n"
        "函数 代理（数值）=内核（数值）。\n变量 输入=readInteger（）。\n"
        "变量 原结果=代理（输入）。\n变量 副本=原结果。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）+错误码（副本）】。\n",
        "41\n",
        "42\n",
        0,
    ),
    (
        "nested function result copy preserves failure",
        "函数 内核（数值）=ifElse（数值==0，失败（17），成功（数值+1））。\n"
        "函数 代理（数值）=内核（数值）。\n变量 输入=readInteger（）。\n"
        "变量 原结果=代理（输入）。\n变量 副本=原结果。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "0\n",
        "",
        17,
    ),
    (
        "static text result selected by function",
        "函数 选择（）=ifElse（1==1，成功（『函数文本』），失败（8））。\n"
        "变量 结果=选择（）。\n程序=【能力=array（『stdout』），输出=结果值（结果）】。\n",
        "",
        "函数文本\n",
        0,
    ),
    (
        "runtime text result selected by function",
        "函数 选择（开关）能力=array（『stdin』）=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "变量 开关=readInteger（）。\n变量 结果=选择（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "1\n动态函数载荷\n",
        "动态函数载荷\n",
        0,
    ),
    (
        "runtime text result failure skips input branch",
        "函数 选择（开关）能力=array（『stdin』）=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "变量 开关=readInteger（）。\n变量 结果=选择（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "0\n",
        "",
        9,
    ),
    (
        "pure function forwards a text result reference",
        "函数 转发（源）=源。\n变量 原结果=成功（readText（））。\n"
        "变量 副本=转发（原结果）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "跨函数文本载荷\n",
        "跨函数文本载荷\n",
        0,
    ),
    (
        "pure function selects between text success and failure",
        "函数 选择（甲，乙，开关）=ifElse（开关==1，甲，乙）。\n"
        "变量 成功值=成功（readText（））。\n变量 开关=readInteger（）。\n"
        "变量 副本=选择（成功值，失败（29），开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "不应输出\n0\n",
        "",
        29,
    ),
    (
        "pure function preserves a failed result reference",
        "函数 转发（源）=源。\n变量 原结果=失败（37）。\n"
        "变量 副本=转发（原结果）。\n"
        "程序=【能力=array（『stdout』），输出=错误码（副本）】。\n",
        "",
        "37\n",
        0,
    ),
    (
        "pure function queries a text result parameter",
        "函数 判（结果）=ifElse（是否失败（结果），失败（1），成功（1））。\n"
        "变量 源=成功（readText（））。\n变量 结果=判（源）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=错误码（结果）】。\n",
        "跨函数状态载荷\n",
        "0\n",
        0,
    ),
    (
        "effectful function initializes a result local",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 结果=ifElse（开关==1，成功（readInteger（）），失败（8））。\n"
        "  返回 结果。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "1\n42\n",
        "42\n",
        0,
    ),
    (
        "effectful result local preserves failure",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 结果=ifElse（开关==1，成功（readInteger（）），失败（8））。\n"
        "  返回 结果。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "0\n",
        "",
        8,
    ),
    (
        "effectful function initializes a text result local",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 甲=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "  变量 乙=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "  变量 丙=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "  返回 丙。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "1\n甲\n乙\n跨函数文本结果局部\n",
        "跨函数文本结果局部\n",
        0,
    ),
    (
        "effectful text result local preserves failure",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 甲=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "  变量 乙=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "  变量 丙=ifElse（开关==1，成功（readText（）），失败（9））。\n"
        "  返回 丙。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "0\n",
        "",
        9,
    ),
    (
        "effectful concatenated text result local",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 结果=ifElse（开关==1，成功（textConcat（readText（），『尾』）），失败（9））。\n"
        "  返回 结果。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "1\n行简\n",
        "行简尾\n",
        0,
    ),
    (
        "effectful concatenated text result local preserves failure",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 结果=ifElse（开关==1，成功（textConcat（readText（），『尾』）），失败（9））。\n"
        "  返回 结果。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "0\n",
        "",
        9,
    ),
    (
        "effectful sliced text result local",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 结果=ifElse（开关==1，成功（substring（readText（），1，3）），失败（9））。\n"
        "  返回 结果。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "1\n行简语言\n",
        "简语\n",
        0,
    ),
    (
        "effectful sliced text result local preserves failure",
        "函数 读取结果（开关）能力=array（『stdin』）【\n"
        "  变量 结果=ifElse（开关==1，成功（substring（readText（），1，3）），失败（9））。\n"
        "  返回 结果。\n"
        "】。\n"
        "变量 开关=readInteger（）。\n变量 结果=读取结果（开关）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "0\n",
        "",
        9,
    ),
]
BLOCK_FUNCTION_CASES = [
    (
        "local sequence and early return",
        (ROOT / "examples" / "p1b-runtime-block-function.serica").read_text(encoding="utf-8"),
        "20\n",
        "41\n",
    ),
    (
        "early branch return",
        "函数 修正（数值）【若（数值>10）【变量 倍数=数值*2。返回 倍数。】。返回 数值+1。】。\n变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=修正（输入）】。\n",
        "21\n",
        "42\n",
    ),
    (
        "conditional local continuation",
        "函数 修正（数值）【若（数值>10）【返回 数值*2。】。返回 数值+1。】。\n变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=修正（输入）】。\n",
        "9\n",
        "10\n",
    ),
    (
        "nested block functions",
        "函数 加一（数值）【变量 结果=数值+1。返回 结果。】。\n函数 双倍（数值）【变量 结果=加一（数值）*2。返回 结果。】。\n变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=双倍（输入）】。\n",
        "20\n",
        "42\n",
    ),
    (
        "same local name in separate branches",
        "函数 选择（数值）【若（数值>0）【变量 调整=数值*2。返回 调整。】否【变量 调整=数值*2+1。返回 调整。】。】。\n变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=选择（输入）】。\n",
        "21\n",
        "42\n",
    ),
    (
        "local parameter replacement",
        "函数 增加（数值）【赋值 数值=数值+1。返回 数值*2。】。\n变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=增加（输入）】。\n",
        "20\n",
        "42\n",
    ),
    (
        "block function effect return",
        "函数 读取倍数（）能力=array（『stdin』）【返回 readInteger（）*2。】。\n程序=【能力=array（『stdout』，『stdin』），输出=读取倍数（）】。\n",
        "21\n",
        "42\n",
    ),
    (
        "block text function local",
        "函数 装饰（文本）【变量 结果=textConcat（文本，『！』）。返回 结果。】。\n变量 输入=readText（）。\n程序=【能力=array（『stdout』，『stdin』），输出=装饰（输入）】。\n",
        "行简\n",
        "行简！\n",
    ),
]
RUNTIME_BLOCK_LOOP_CASES = [
    (
        "runtime block loop accumulator",
        (ROOT / "examples" / "p1b-runtime-block-loop.serica").read_text(encoding="utf-8"),
        "10\n",
        "55\n",
    ),
    (
        "runtime block loop zero iterations",
        (ROOT / "examples" / "p1b-runtime-block-loop.serica").read_text(encoding="utf-8"),
        "0\n",
        "0\n",
    ),
    (
        "runtime block loop read integer as left addend",
        "函数 读取和（次数）能力=array（『stdin』）【\n"
        "  变量 合计=0。\n"
        "  变量 索引=0。\n"
        "  凡（索引<次数）【赋值 合计=readInteger（）+合计。赋值 索引=索引+1。】。\n"
        "  返回 合计。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=读取和（输入）】。\n",
        "3\n1\n2\n3\n",
        "6\n",
    ),
    (
        "runtime block loop two parameters",
        "函数 阶乘循环（上限，结果）【凡（上限>1）【赋值 结果=结果*上限。赋值 上限=上限-1。】。返回 结果。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=阶乘循环（输入，1）】。\n",
        "6\n",
        "720\n",
    ),
    (
        "runtime block two sequential loops",
        "函数 累计两段（上限）【变量 当前=0。变量 合计=0。"
        "凡（当前<上限）【赋值 当前=当前+1。赋值 合计=合计+当前。】。"
        "凡（当前>0）【赋值 当前=当前-1。赋值 合计=合计+1。】。返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=累计两段（输入）】。\n",
        "5\n",
        "20\n",
    ),
    (
        "runtime block loop with static false condition",
        "函数 保持（数值）【凡（0<0）【赋值 数值=数值+1。】。返回 数值。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=保持（输入）】。\n",
        "42\n",
        "42\n",
    ),
    (
        "runtime block loop break and continue",
        (ROOT / "examples" / "p1b-runtime-block-loop-control.serica").read_text(encoding="utf-8"),
        "10\n",
        "9\n",
    ),
    (
        "runtime block nested loops",
        "函数 三角（上限）【变量 外层=0。变量 合计=0。变量 内层=0。"
        "凡（外层<上限）【赋值 外层=外层+1。赋值 内层=0。"
        "凡（内层<外层）【赋值 内层=内层+1。若（内层==2）【继续。】。赋值 合计=合计+内层。】。"
        "】。返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=三角（输入）】。\n",
        "3\n",
        "6\n",
    ),
    (
        "runtime block loop if else",
        "函数 加权（上限）【变量 当前=上限。变量 合计=0。"
        "凡（当前>0）【若（当前%2==0）【赋值 合计=合计+当前。】否【赋值 合计=合计+当前*10。】。赋值 当前=当前-1。】。"
        "返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=加权（输入）】。\n",
        "2\n",
        "12\n",
    ),
    (
        "runtime block nested loop break",
        "函数 前三个（上限）【变量 外层=0。变量 合计=0。变量 内层=0。"
        "凡（外层<上限）【赋值 外层=外层+1。赋值 内层=0。"
        "凡（内层<10）【赋值 内层=内层+1。若（内层==3）【跳出。】。赋值 合计=合计+1。】。"
        "】。返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=前三个（输入）】。\n",
        "4\n",
        "8\n",
    ),
    (
        "runtime block loop local initialized per iteration",
        (ROOT / "examples" / "p1b-runtime-loop-local.serica").read_text(encoding="utf-8"),
        "5\n",
        "15\n",
    ),
    (
        "runtime block loop branch locals",
        "函数 分支局部（上限）【变量 当前=0。变量 合计=0。"
        "凡（当前<上限）【赋值 当前=当前+1。"
        "若（当前%2==0）【变量 偶数=当前*2。赋值 合计=合计+偶数。】否【变量 奇数=当前*2+1。赋值 合计=合计+奇数。】。"
        "】。返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=分支局部（输入）】。\n",
        "3\n",
        "14\n",
    ),
    (
        "runtime block nested loop locals",
        "函数 嵌套局部（上限）【变量 外层=0。变量 合计=0。"
        "凡（外层<上限）【变量 行=外层+1。变量 内层=0。"
        "凡（内层<行）【变量 单元=行+内层。赋值 内层=内层+1。赋值 合计=合计+单元。】。"
        "赋值 外层=外层+1。】。返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=嵌套局部（输入）】。\n",
        "3\n",
        "18\n",
    ),
    (
        "runtime block loop early return",
        (ROOT / "examples" / "p1b-runtime-loop-return.serica").read_text(encoding="utf-8"),
        "5\n",
        "30\n",
    ),
    (
        "runtime block loop early return fallthrough",
        (ROOT / "examples" / "p1b-runtime-loop-return.serica").read_text(encoding="utf-8"),
        "2\n",
        "-1\n",
    ),
    (
        "runtime block nested loop return",
        "函数 嵌套命中（上限）【变量 外层=0。"
        "凡（外层<上限）【赋值 外层=外层+1。变量 内层=0。"
        "凡（内层<上限）【赋值 内层=内层+1。若（内层==2）【返回 外层*10+内层。】。】。"
        "】。返回 0。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=嵌套命中（输入）】。\n",
        "3\n",
        "12\n",
    ),
    (
        "runtime block top-level guard return",
        (ROOT / "examples" / "p1b-runtime-top-level-if.serica").read_text(encoding="utf-8"),
        "-1\n",
        "0\n",
    ),
    (
        "runtime block top-level guard fallthrough",
        (ROOT / "examples" / "p1b-runtime-top-level-if.serica").read_text(encoding="utf-8"),
        "4\n",
        "10\n",
    ),
    (
        "runtime block local after loop",
        "函数 循环后局部（上限）【变量 当前=上限。凡（当前>0）【赋值 当前=当前-1。】。"
        "变量 结果=当前+1。返回 结果。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=循环后局部（输入）】。\n",
        "0\n",
        "1\n",
    ),
    (
        "runtime block branch local before loop",
        "函数 分支起点（上限）【变量 起点=0。"
        "若（上限>3）【变量 高位=3。赋值 起点=高位。】否【变量 低位=上限。赋值 起点=低位。】。"
        "变量 当前=0。变量 合计=0。凡（当前<起点）【赋值 当前=当前+1。赋值 合计=合计+当前。】。返回 合计。】。\n"
        "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』，『stdin』），输出=分支起点（输入）】。\n",
        "5\n",
        "6\n",
    ),
]
RUNTIME_FUNCTION_RESULT_RECOVERY_CASES = [
    (
        "block function forwards observed result to caller",
        "函数 内层（数值）=ifElse（数值<0，失败（31），成功（数值+1））。\n"
        "函数 外层（数值）【\n"
        "    变量 中途=内层（数值）。\n"
        "    若（是否失败（中途））【返回 失败（错误码（中途））。】。\n"
        "    返回 中途。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=外层（输入）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "5\n",
        "6\n",
        0,
    ),
    (
        "result state parameter receives inline result expression",
        "函数 内层（数值）=ifElse（数值<0，失败（31），成功（数值+1））。\n"
        "函数 转发（源）【\n"
        "    若（是否失败（源））【返回 源。】。\n"
        "    返回 源。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=转发（内层（输入））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（是否失败（结果），错误码（结果），结果值（结果））】。\n",
        "5\n",
        "6\n",
        0,
    ),
    (
        "result state parameter preserves inline failure",
        "函数 内层（数值）=ifElse（数值<0，失败（31），成功（数值+1））。\n"
        "函数 转发（源）【\n"
        "    若（是否失败（源））【返回 源。】。\n"
        "    返回 源。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=转发（内层（输入））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（是否失败（结果），错误码（结果），结果值（结果））】。\n",
        "-1\n",
        "31\n",
        0,
    ),
    (
        "result state parameter rejects inline text result payload",
        "函数 内层（状态）=ifElse（状态==1，成功（『甲』），失败（9））。\n"
        "函数 转发（源）【\n"
        "    若（是否失败（源））【返回 源。】。\n"
        "    返回 源。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=转发（内层（输入））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=错误码（结果）】。\n",
        "1\n",
        "S-3023",
        "1\n",
        0,
        "S-3023",
        "check",
    ),
    (
        "mixed integer and result parameter forwarding success",
        "函数 内层（数值）=ifElse（数值<0，失败（31），成功（数值+1））。\n"
        "函数 转发（甲，源）【\n"
        "    若（是否失败（源））【返回 失败（错误码（源）+甲）。】。\n"
        "    返回 成功（结果值（源）+甲）。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=转发（5，内层（输入））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（是否失败（结果），错误码（结果），结果值（结果））】。\n",
        "3\n",
        "9\n",
        0,
    ),
    (
        "mixed integer and result parameter forwarding failure",
        "函数 内层（数值）=ifElse（数值<0，失败（31），成功（数值+1））。\n"
        "函数 转发（甲，源）【\n"
        "    若（是否失败（源））【返回 失败（错误码（源）+甲）。】。\n"
        "    返回 成功（结果值（源）+甲）。\n"
        "】。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=转发（5，内层（输入））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（是否失败（结果），错误码（结果），结果值（结果））】。\n",
        "-1\n",
        "36\n",
        0,
    ),
    (
        "block function local result recovery success",
        (ROOT / "examples" / "p1b-function-result-recovery.serica").read_text(encoding="utf-8"),
        "5\n",
        "6\n",
    ),
    (
        "block function local result recovery failure",
        (ROOT / "examples" / "p1b-function-result-recovery.serica").read_text(encoding="utf-8"),
        "-1\n",
        "31\n",
    ),
]
OWNED_TEXT_RETURN_CASES = [
    (
        "owned text return from copied parameter",
        (ROOT / "examples" / "p1b-runtime-text-return.serica").read_text(
            encoding="utf-8"
        ),
        "行简\n",
        "行简\n",
    ),
    (
        "owned text return grows after local replacement",
        "函数 改写（来源）【变量 副本=来源。赋值 副本=textConcat（副本，『尾』）。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=改写（输入）】。\n",
        "x" * 63 + "\n",
        "x" * 63 + "尾\n",
    ),
    (
        "owned text return from substring",
        "函数 首段（来源）【变量 副本=substring（来源，0，2）。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=首段（输入）】。\n",
        "甲乙丙\n",
        "甲乙\n",
    ),
    (
        "owned supplementary text return cleans other local",
        "函数 转交（来源）【变量 保留=来源。变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=转交（输入）】。\n",
        "𠮷野家\n",
        "𠮷野家\n",
    ),
    (
        "owned empty text return",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=欢迎（输入）】。\n",
        "\n",
        "\n",
    ),
]
OWNED_TEXT_RETURN_STORAGE_CASES = [
    (
        "owned text return borrowed equality",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（equals（欢迎（输入），『行简』），1，0）】。\n",
        "行简\n",
        "1\n",
        0,
    ),
    (
        "owned text return borrowed inequality",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（notEquals（欢迎（输入），『行简』），1，0）】。\n",
        "其他\n",
        "1\n",
        0,
    ),
    (
        "owned text return length operand",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=textLength（欢迎（输入））】。\n",
        "行简\n",
        "2\n",
    ),
    (
        "owned supplementary text return length operand",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=textLength（欢迎（输入））】。\n",
        "𠮷野家\n",
        "3\n",
    ),
    (
        "owned text return initializes a variable",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "变量 结果=欢迎（输入）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果】。\n",
        "行简\n",
        "行简\n",
    ),
    (
        "owned text return replaces a variable",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "变量 其他=readText（）。\n"
        "变量 结果=欢迎（输入）。\n"
        "赋值 结果=欢迎（其他）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果】。\n",
        "甲\n乙\n",
        "乙\n",
    ),
    (
        "owned empty text return initializes a variable",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "变量 结果=欢迎（输入）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果】。\n",
        "\n",
        "\n",
    ),
    (
        "owned supplementary text return initializes a variable",
        "函数 转交（来源）【变量 保留=来源。变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "变量 结果=转交（输入）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果】。\n",
        "𠮷野家\n",
        "𠮷野家\n",
    ),
    (
        "owned long text return initializes and replaces a variable",
        "函数 改写（来源）【变量 副本=来源。赋值 副本=textConcat（副本，『尾』）。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "变量 结果=改写（输入）。\n"
        "赋值 结果=改写（输入）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果】。\n",
        "x" * 63 + "\n",
        "x" * 63 + "尾\n",
    ),
    (
        "owned text return repeatedly replaces a variable",
        "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
        "变量 输入=readText（）。\n"
        "变量 结果=欢迎（输入）。\n"
        "赋值 结果=欢迎（输入）。\n"
        "赋值 结果=欢迎（输入）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果】。\n",
        "行简\n",
        "行简\n",
    ),
]
RUNTIME_TEXT_PARAMETER_CASES = [
    (
        "copied text parameter length",
        (ROOT / "examples" / "p1b-runtime-text-parameter-copy.serica").read_text(
            encoding="utf-8"
        ),
        "行简\n",
        "2\n",
    ),
    (
        "copied text parameter comparison",
        "函数 匹配（来源）【变量 副本=来源。若（equals（副本，『行简』））【返回 1。】。返回 0。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=匹配（输入）】。\n",
        "行简\n",
        "1\n",
    ),
    (
        "copied text parameter grows through concatenation",
        "函数 增长（来源）【变量 副本=来源。赋值 副本=textConcat（副本，『x』）。"
        "返回 textLength（副本）。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=增长（输入）】。\n",
        "x" * 63 + "\n",
        "64\n",
    ),
    (
        "independent copied text locals survive source reassignment",
        "函数 独立性（来源）【变量 甲=来源。变量 乙=甲。"
        "赋值 甲=textConcat（甲，『！』）。返回 textLength（乙）*10+textLength（甲）。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=独立性（输入）】。\n",
        "行简\n",
        "23\n",
    ),
    (
        "copied text parameter substring",
        "函数 首字（来源）【变量 副本=substring（来源，0，1）。返回 textLength（副本）。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=首字（输入）】。\n",
        "行简\n",
        "1\n",
    ),
    (
        "copied supplementary scalar text parameter",
        "函数 副本长度（来源）【变量 副本=来源。返回 textLength（副本）。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=副本长度（输入）】。\n",
        "𠮷野家\n",
        "3\n",
    ),
    (
        "early return cleans initialized copied text",
        "函数 提前返回（来源）【变量 甲=来源。"
        "若（textLength（甲）>=0）【返回 textLength（甲）。】。"
        "变量 未初始化=来源。返回 textLength（未初始化）。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=提前返回（输入）】。\n",
        "行简\n",
        "2\n",
    ),
    (
        "runtime text parameter equality and ordering",
        (ROOT / "examples" / "p1b-runtime-text-parameter-comparison.serica").read_text(encoding="utf-8"),
        "行简\n",
        "1001111\n",
    ),
    (
        "runtime text parameter length",
        (ROOT / "examples" / "p1b-runtime-text-parameter.serica").read_text(encoding="utf-8"),
        "甲乙\n",
        "5\n",
    ),
    (
        "static text parameter length",
        "函数 文本长度（来源）【返回 textLength（来源）。】。\n"
        "程序=【能力=array（『stdout』），输出=文本长度（『行简』）】。\n",
        "",
        "2\n",
    ),
    (
        "direct readText parameter",
        "函数 文本长度（来源）【返回 textLength（来源）。】。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=文本长度（readText（））】。\n",
        "甲乙\n",
        "2\n",
    ),
    (
        "two runtime text parameters",
        "函数 两段长度（甲，乙）【返回 textLength（甲）+textLength（乙）。】。\n"
        "变量 第一=readText（）。\n变量 第二=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=两段长度（第一，第二）】。\n",
        "甲乙\nA\n",
        "3\n",
    ),
    (
        "concatenated runtime text parameter",
        "函数 文本长度（来源）【返回 textLength（来源）。】。\n"
        "变量 左=readText（）。\n变量 右=readText（）。\n变量 文本=textConcat（左，右）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=文本长度（文本）】。\n",
        "行\n简\n",
        "2\n",
    ),
    (
        "integer text integer register order",
        "函数 混合（首位，来源，末位）【返回 首位*100+textLength（来源）*10+末位。】。\n"
        "变量 输入=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=混合（1，输入，2）】。\n",
        "甲乙\n",
        "122\n",
    ),
    (
        "full eight flattened parameter registers",
        "函数 预算（甲，乙，一，二，三，四）【"
        "返回 textLength（甲）*100000+textLength（乙）*10000+一*1000+二*100+三*10+四。】。\n"
        "变量 甲=readText（）。\n变量 乙=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=预算（甲，乙，1，2，3，4）】。\n",
        "甲乙\nA\n",
        "211234\n",
    ),
    (
        "static strings passed to two text parameters",
        "函数 相同（左，右）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（左，右））【返回 1。】。返回 0。】。\n"
        "变量 占位=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=相同（『行简』，『行简』）+占位】。\n",
        "0\n",
        "1\n",
    ),
    (
        "direct readText argument compared with static text",
        "函数 相同（左）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（左，『行简』））【返回 1。】。返回 0。】。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=相同（readText（））】。\n",
        "行简\n",
        "1\n",
    ),
    (
        "two runtime text variables compare by parameter equality",
        "函数 相同（左，右）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（左，右））【返回 1。】。返回 0。】。\n"
        "变量 第一=readText（）。变量 第二=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=相同（第一，第二）】。\n",
        "行简\n行简\n",
        "1\n",
    ),
    (
        "same text parameter in inclusive ordering operands",
        "函数 自身（来源）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（来源，『行简』））【赋值 守卫=守卫+0。】。"
        "若（来源<=来源 且 来源>=来源）【返回 1。】。返回 0。】。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=自身（readText（））】。\n",
        "行简\n",
        "1\n",
    ),
    (
        "concatenated text argument compares with runtime parameter",
        "函数 相同（左，右）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（左，右））【返回 1。】。返回 0。】。\n"
        "变量 第一=readText（）。变量 第二=readText（）。\n"
        "程序=【能力=array（『stdout』，『stdin』，『array.write』），输出=相同（textConcat（第一，『』），第二）】。\n",
        "行简\n行简\n",
        "1\n",
    ),
    (
        "supplementary scalar text parameter ordering",
        "函数 小于（左，右）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（左，『行简』））【赋值 守卫=守卫+0。】。"
        "若（左<右）【返回 1。】。返回 0。】。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=小于（readText（），『🟿』）】。\n",
        "😀\n",
        "1\n",
    ),
    (
        "integer equality parameters remain integer",
        "函数 相同（左，右）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
        "若（equals（左，右））【返回 1。】。返回 0。】。\n"
        "变量 第一=readInteger（）。变量 第二=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=相同（第一，第二）】。\n",
        "7\n7\n",
        "1\n",
    ),
]
RUNTIME_FUNCTION_ABI_CASES = [
    (
        "recursive factorial call frame",
        RECURSIVE_FUNCTION_FIXTURE,
        "5\n",
        "120\n",
    ),
    (
        "recursive fibonacci nested calls",
        "函数 斐波那契（数值）=ifElse（数值<=1，数值，斐波那契（数值-1）+斐波那契（数值-2））。\n"
        "变量 输入=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=斐波那契（输入）】。\n",
        "8\n",
        "21\n",
    ),
    (
        "mutual recursion call cycle",
        "函数 偶数（数值）=ifElse（数值==0，1，奇数（数值-1））。\n"
        "函数 奇数（数值）=ifElse（数值==0，0，偶数（数值-1））。\n"
        "变量 输入=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=偶数（输入）*10+奇数（输入）】。\n",
        "7\n",
        "1\n",
    ),
    (
        "recursive call stack depth 200",
        "函数 累计（数值）=ifElse（数值<=0，0，数值+累计（数值-1））。\n"
        "变量 输入=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=累计（输入）】。\n",
        "200\n",
        "20100\n",
    ),
    (
        "recursive two argument register ABI",
        "函数 乘方（底数，指数）=ifElse（指数==0，1，底数*乘方（底数，指数-1））。\n"
        "变量 基数=readInteger（）。\n变量 指数=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=乘方（基数，指数）】。\n",
        "2\n10\n",
        "1024\n",
    ),
    (
        "recursive function global constant",
        "常量 步长=3。\n"
        "函数 累计（数值）=ifElse（数值<=0，0，步长+累计（数值-步长））。\n"
        "变量 输入=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=累计（输入）】。\n",
        "9\n",
        "9\n",
    ),
]
RUNTIME_RESULT_ABI_CASES = [
    (
        "recursive result function success",
        (ROOT / "examples" / "p1b-runtime-recursive-result.serica").read_text(encoding="utf-8"),
        "5\n",
        "15\n",
        0,
    ),
    (
        "recursive result function failure propagation",
        (ROOT / "examples" / "p1b-runtime-recursive-result.serica").read_text(encoding="utf-8"),
        "-1\n",
        "",
        31,
    ),
    (
        "recursive result function depth 200",
        (ROOT / "examples" / "p1b-runtime-recursive-result.serica").read_text(encoding="utf-8"),
        "200\n",
        "20100\n",
        0,
    ),
    (
        "recursive result failure state without exit",
        "函数 累计（数值，结果）=ifElse（数值<0，失败（31），ifElse（数值==0，成功（结果），累计（数值-1，结果+数值）））。\n"
        "变量 输入=readInteger（）。\n"
        "变量 结果=累计（输入，0）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=错误码（结果）*100+ifElse（是否失败（结果），1，0）】。\n",
        "-1\n",
        "3101\n",
    ),
    (
        "mutually recursive result function success",
        "函数 偶数（数值）=ifElse（数值==0，成功（1），奇数（数值-1））。\n"
        "函数 奇数（数值）=ifElse（数值==0，失败（17），偶数（数值-1））。\n"
        "变量 结果=偶数（readInteger（））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "6\n",
        "1\n",
    ),
    (
        "mutually recursive result function failure",
        "函数 偶数（数值）=ifElse（数值==0，成功（1），奇数（数值-1））。\n"
        "函数 奇数（数值）=ifElse（数值==0，失败（17），偶数（数值-1））。\n"
        "变量 结果=偶数（readInteger（））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "5\n",
        "",
        17,
    ),
    (
        "recursive result state parameter",
        "函数 累计（次数，结果）=ifElse（次数<=0，结果，累计（次数-1，结果））。\n"
        "变量 源=失败（7）。\n"
        "变量 结果=累计（1，源）。\n"
        "程序=【能力=array（『stdout』），输出=ifElse（是否失败（结果），错误码（结果），0）】。\n",
        "",
        "7\n",
    ),
    (
        "mutually recursive result state parameter",
        "函数 甲（次数，结果）=ifElse（次数<=0，结果，乙（次数-1，结果））。\n"
        "函数 乙（次数，结果）=ifElse（次数<=0，结果，甲（次数-1，结果））。\n"
        "变量 源=失败（9）。\n"
        "变量 结果=甲（2，源）。\n"
        "程序=【能力=array（『stdout』），输出=ifElse（是否失败（结果），错误码（结果），0）】。\n",
        "",
        "9\n",
    ),

]

RUNTIME_TEXT_RESULT_ABI_CASES = [
    (
        "recursive result function static text success",
        (ROOT / "examples" / "p1b-runtime-recursive-text-result.serica").read_text(encoding="utf-8"),
        "3\n",
        "到达零\n",
        0,
    ),
    (
        "recursive result function static text failure",
        "函数 校验（数值）=ifElse（数值<0，失败（27），ifElse（数值==0，成功（『合法』），校验（数值-1）））。\n"
        "变量 结果=校验（readInteger（））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "-1\n",
        "",
        27,
    ),
    (
        "mutually recursive static text result success",
        "函数 偶（数值）=ifElse（数值==0，成功（『偶』），奇（数值-1））。\n"
        "函数 奇（数值）=ifElse（数值==0，成功（『奇』），偶（数值-1））。\n"
        "变量 结果=偶（readInteger（））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "5\n",
        "奇\n",
    ),
    (
        "mutually recursive static text result failure",
        "函数 偶（数值）=ifElse（数值<0，失败（29），ifElse（数值==0，成功（『偶』），奇（数值-1）））。\n"
        "函数 奇（数值）=ifElse（数值<0，失败（29），ifElse（数值==0，成功（『奇』），偶（数值-1）））。\n"
        "变量 结果=偶（readInteger（））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "-2\n",
        "",
        29,
    ),
    (
        "recursive text result deep copy into text variable",
        "函数 倒数（数值）=ifElse（数值==0，成功（『零』），ifElse（数值==1，成功（『一』），倒数（数值-1）））。\n"
        "变量 源=倒数（readInteger（））。\n"
        "变量 目标=结果值（源）。\n"
        "赋值 源=倒数（readInteger（））。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=目标】。\n",
        "1\n0\n",
        "一\n",
    ),
]
DYNAMIC_TEXT_RESULT_CASES = [
    (
        "runtime text result output",
        (ROOT / "examples" / "p1b-runtime-text-result-dynamic.serica").read_text(encoding="utf-8"),
        "行简动态载荷\n",
        "行简动态载荷\n",
        0,
    ),
    (
        "runtime text result array element",
        "变量 项=array（『甲』，『乙』）。\n变量 索引=readInteger（）。\n变量 结果=成功（arrayGet（项，索引））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "0\n",
        "甲\n",
        0,
    ),
    (
        "runtime text result array element upper index",
        "变量 项=array（『甲』，『乙』）。\n变量 索引=readInteger（）。\n变量 结果=成功（arrayGet（项，索引））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "1\n",
        "乙\n",
        0,
    ),
    (
        "runtime text result array element out of bounds",
        "变量 项=array（『甲』，『乙』）。\n变量 索引=readInteger（）。\n变量 结果=成功（arrayGet（项，索引））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "2\n",
        "",
        1,
    ),
    (
        "runtime text result array element negative index",
        "变量 项=array（『甲』，『乙』）。\n变量 索引=readInteger（）。\n变量 结果=成功（arrayGet（项，索引））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "-1\n",
        "",
        1,
    ),
    (
        "runtime text result supplementary plane",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "𠮷野家\n",
        "𠮷野家\n",
        0,
    ),
    (
        "runtime text result reassignment",
        "变量 结果=成功（readText（））。\n赋值 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "旧载荷\n新载荷\n",
        "新载荷\n",
        0,
    ),
    (
        "runtime text result failure replacement",
        "变量 结果=成功（readText（））。\n赋值 结果=失败（9）。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "不应输出\n",
        "",
        9,
    ),
    (
        "runtime text result copy",
        "变量 原结果=成功（readText（））。\n变量 副本=原结果。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "复制载荷\n",
        "复制载荷\n",
        0,
    ),
    (
        "runtime text result copy independent after source reassignment",
        (ROOT / "examples" / "p1b-runtime-text-result-copy.serica").read_text(encoding="utf-8"),
        "旧载荷\n新载荷\n",
        "旧载荷\n",
        0,
    ),
    (
        "runtime text result copy independent after source failure replacement",
        "变量 原结果=成功（readText（））。\n变量 副本=原结果。\n赋值 原结果=失败（9）。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "保留载荷\n",
        "保留载荷\n",
        0,
    ),
    (
        "runtime text result copy chain",
        "变量 原结果=成功（readText（））。\n变量 副本=原结果。\n变量 再副本=副本。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（再副本）】。\n",
        "连续复制\n",
        "连续复制\n",
        0,
    ),
    (
        "runtime text result copy preserves failure",
        "变量 原结果=成功（readText（））。\n赋值 原结果=失败（19）。\n变量 副本=原结果。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "不应输出\n",
        "",
        19,
    ),
    (
        "runtime text result long UTF-8 copy",
        "变量 原结果=成功（readText（））。\n变量 副本=原结果。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（副本）】。\n",
        "行简" * 2048 + "\n",
        "行简" * 2048 + "\n",
        0,
    ),
    (
        "runtime text result success status",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=是否失败（结果）】。\n",
        "状态\n",
        "0\n",
        0,
    ),
    (
        "runtime text result empty line",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "\n",
        "\n",
        0,
    ),
    (
        "runtime text result long UTF-8 payload",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "行简" * 2048 + "\n",
        "行简" * 2048 + "\n",
        0,
    ),
    (
        "runtime text result code point length",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=textLength（结果值（结果））】。\n",
        "𠮷野家\n",
        "3\n",
        0,
    ),
    (
        "runtime text result concatenation",
        (ROOT / "examples" / "p1b-runtime-text-result-operations.serica").read_text(encoding="utf-8"),
        "行简\n",
        "行简！\n",
        0,
    ),
    (
        "runtime text result substring",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=substring（结果值（结果），1，3）】。\n",
        "行简语言\n",
        "简语\n",
        0,
    ),
    (
        "runtime text result equality",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=ifElse（equals（结果值（结果），『行简』），1，0）】。\n",
        "行简\n",
        "1\n",
        0,
    ),
    (
        "runtime text result ordering",
        "变量 结果=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=ifElse（结果值（结果）>『行简』，1，0）】。\n",
        "行简语言\n",
        "1\n",
        0,
    ),
    (
        "runtime text result concatenation payload",
        (ROOT / "examples" / "p1b-runtime-text-result-payload.serica").read_text(encoding="utf-8"),
        "行简\n",
        "行简！\n",
        0,
    ),
    (
        "runtime text result substring payload",
        "变量 文本=readText（）。\n变量 结果=成功（substring（文本，1，3））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "行简语言\n",
        "简语\n",
        0,
    ),
    (
        "runtime text result long concatenation payload",
        "变量 文本=readText（）。\n变量 结果=成功（textConcat（文本，『尾』））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "行简" * 100 + "\n",
        "行简" * 100 + "尾\n",
        0,
    ),
    (
        "runtime text result self concatenation payload",
        "变量 结果=成功（readText（））。\n赋值 结果=成功（textConcat（『！』，结果值（结果）））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "行简\n",
        "！行简\n",
        0,
    ),
    (
        "runtime text result concatenation payload from result",
        "变量 源=成功（readText（））。\n变量 结果=成功（textConcat（结果值（源），『！』））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "行简\n",
        "行简！\n",
        0,
    ),
    (
        "runtime text result substring payload from result",
        "变量 源=成功（readText（））。\n变量 结果=成功（substring（结果值（源），1，3））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "行简语言\n",
        "简语\n",
        0,
    ),
    (
        "runtime text result payload source failure propagation",
        "变量 源=成功（readText（））。\n赋值 源=失败（21）。\n变量 结果=成功（textConcat（结果值（源），『！』））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "不应输出\n",
        "",
        21,
    ),
    (
        "runtime text result reassigned to concatenation payload",
        "变量 文本=readText（）。\n变量 结果=成功（textConcat（文本，『一』））。\n变量 后文=readText（）。\n赋值 结果=成功（textConcat（后文，『二』））。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n",
        "前文\n后文\n",
        "后文二\n",
        0,
    ),
    (
        "runtime text result initializes text variable",
        (ROOT / "examples" / "p1b-runtime-text-result-storage.serica").read_text(encoding="utf-8"),
        "前文\n",
        "前文\n",
        0,
    ),
    (
        "runtime text result assigns text variable",
        "变量 文本=readText（）。\n变量 源=成功（readText（））。\n赋值 文本=结果值（源）。\n赋值 源=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』），输出=文本】。\n",
        "旧文\n新文\n后文\n",
        "新文\n",
        0,
    ),
    (
        "runtime concatenation result copies to text variable",
        "变量 文本=成功（textConcat（readText（），『！』））。\n变量 目标=结果值（文本）。\n程序=【能力=array（『stdout』，『stdin』），输出=目标】。\n",
        "行简\n",
        "行简！\n",
        0,
    ),
    (
        "runtime text result long UTF-8 copies to text variable",
        "变量 源=成功（readText（））。\n变量 文本=结果值（源）。\n程序=【能力=array（『stdout』，『stdin』），输出=文本】。\n",
        "行简" * 2048 + "\n",
        "行简" * 2048 + "\n",
        0,
    ),
    (
        "failed text result cannot initialize variable",
        "变量 源=成功（readText（））。\n赋值 源=失败（23）。\n变量 文本=结果值（源）。\n程序=【能力=array（『stdout』，『stdin』），输出=文本】。\n",
        "不应输出\n",
        "",
        23,
    ),
    (
        "runtime text result initializes text array element",
        "变量 源=成功（readText（））。\n变量 数组=array（结果值（源），『尾』）。\n赋值 源=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』，『array.write』），输出=arrayGet（数组，0）】。\n",
        "前文\n后文\n",
        "前文\n",
        0,
    ),
    (
        "runtime text result pushes text array element",
        "变量 源=成功（readText（））。\n变量 数组=array（『头』）。\n赋值 数组=arrayPush（数组，结果值（源））。\n程序=【能力=array（『stdout』，『stdin』，『array.write』），输出=arrayGet（数组，1）】。\n",
        "载荷\n",
        "载荷\n",
        0,
    ),
    (
        "runtime text result sets text array element",
        "变量 源=成功（readText（））。\n变量 数组=array（『旧值』，『尾』）。\n赋值 数组=arraySet（数组，0，结果值（源））。\n赋值 源=成功（readText（））。\n程序=【能力=array（『stdout』，『stdin』，『array.write』），输出=arrayGet（数组，0）】。\n",
        "新值\n后文\n",
        "新值\n",
        0,
    ),
    (
        "failed text result cannot push array element",
        "变量 源=成功（readText（））。\n赋值 源=失败（24）。\n变量 数组=array（『头』）。\n赋值 数组=arrayPush（数组，结果值（源））。\n程序=【能力=array（『stdout』，『stdin』，『array.write』），输出=arrayGet（数组，0）】。\n",
        "不应输出\n",
        "",
        24,
    ),
    (
        "runtime text result operation failure propagation",
        "变量 结果=成功（readText（））。\n赋值 结果=失败（17）。\n程序=【能力=array（『stdout』，『stdin』），输出=textLength（结果值（结果））】。\n",
        "不应输出\n",
        "",
        17,
    ),
]
TEXT_RESULT_CASES = [
    (
        "static text result code point length",
        "变量 结果=成功（『𠮷野』）。\n程序=【输出=textLength（结果值（结果））】。\n",
        "2\n",
    ),
    (
        "static text result concatenation",
        "变量 结果=成功（『行简』）。\n程序=【输出=textConcat（结果值（结果），『！』）】。\n",
        "行简！\n",
    ),
    (
        "static text result substring",
        "变量 结果=成功（『行简语言』）。\n程序=【输出=substring（结果值（结果），1，3）】。\n",
        "简语\n",
    ),
    (
        "static text result ordering",
        "变量 结果=成功（『行简语言』）。\n程序=【输出=ifElse（结果值（结果）>『行简』，1，0）】。\n",
        "1\n",
    ),
    (
        "static text result output",
        "变量 结果=成功（『行简』）。\n程序=【输出=结果值（结果）】。\n",
        "行简\n",
    ),
    (
        "compile-time expression text result",
        "变量 结果=成功（『行』+『简』）。\n程序=【输出=结果值（结果）】。\n",
        "行简\n",
    ),
    (
        "copied text result",
        "变量 原结果=成功（『复制载荷』）。\n变量 副本=原结果。\n程序=【输出=结果值（副本）】。\n",
        "复制载荷\n",
    ),
    (
        "reassigned text result",
        "变量 结果=成功（『旧』）。\n赋值 结果=成功（『新载荷』）。\n程序=【输出=结果值（结果）】。\n",
        "新载荷\n",
    ),
    (
        "text result failure status",
        "变量 结果=成功（『状态』）。\n程序=【输出=是否失败（结果）】。\n",
        "0\n",
    ),
    (
        "supplementary plane text result",
        "变量 结果=成功（『𠮷』）。\n程序=【输出=结果值（结果）】。\n",
        "𠮷\n",
    ),
]
DYNAMIC_FAILURES = [
    ("runtime add overflow", "程序=【输出=readInteger（）+1】。\n", "9223372036854775807\n"),
    ("runtime subtraction overflow", "程序=【输出=readInteger（）-1】。\n", "-9223372036854775808\n"),
    ("runtime multiplication overflow", "程序=【输出=readInteger（）*2】。\n", "9223372036854775807\n"),
    ("runtime division zero", "程序=【输出=readInteger（）/readInteger（）】。\n", "7\n0\n"),
    ("runtime division overflow", "程序=【输出=readInteger（）/readInteger（）】。\n", "-9223372036854775808\n-1\n"),
    ("runtime remainder zero", "程序=【输出=readInteger（）%readInteger（）】。\n", "7\n0\n"),
    ("runtime negative shift", "程序=【输出=1<<readInteger（）】。\n", "-1\n"),
    ("runtime large shift", "程序=【输出=1>>readInteger（）】。\n", "64\n"),
    ("runtime left shift overflow", "程序=【输出=1<<readInteger（）】。\n", "63\n"),
    ("runtime negation overflow", "程序=【输出=-readInteger（）】。\n", "-9223372036854775808\n"),
    (
        "runtime checked arithmetic in if body",
        "变量 数值=readInteger（）。\n若（数值==0）【赋值 数值=readInteger（）+1。】。\n程序=【输出=数值】。\n",
        "0\n9223372036854775807\n",
    ),
    ("runtime array index above length", "变量 数组=array（1，2，3）。\n程序=【输出=arrayGet（数组，readInteger（））】。\n", "3\n"),
    ("runtime array negative index", "变量 数组=array（1，2，3）。\n程序=【输出=arrayGet（数组，readInteger（））】。\n", "-1\n"),
    ("array set positive index beyond current length", "变量 数组=array（1，2）。\n赋值 数组=arraySet（数组，2，9）。\n程序=【输出=1】。\n", ""),
    ("runtime boolean array index above length", "变量 标志=array（1<2）。程序=【输出=ifElse（arrayGet（标志，readInteger（）），42，-1）】。\n", "1\n"),
    ("runtime boolean array negative index", "变量 标志=array（1<2）。程序=【输出=ifElse（arrayGet（标志，readInteger（）），42，-1）】。\n", "-1\n"),
]
RUNTIME_TEXT_CASES = [
    ("ASCII line", b"Serica\n", b"Serica\n"),
    ("UTF-8 Chinese line", "行简文本\n".encode("utf-8"), "行简文本\n".encode("utf-8")),
    ("CRLF line", b"line\r\n", b"line\n"),
    ("empty line", b"\n", b"\n"),
    ("EOF terminated line", b"tail", b"tail\n"),
    ("maximum 63-byte line", b"a" * 63 + b"\n", b"a" * 63 + b"\n"),
    ("lowest three-byte scalar", "\u0800\n".encode("utf-8"), "\u0800\n".encode("utf-8")),
    ("highest pre-surrogate scalar", "\ud7ff\n".encode("utf-8"), "\ud7ff\n".encode("utf-8")),
    ("lowest four-byte scalar", "\U00010000\n".encode("utf-8"), "\U00010000\n".encode("utf-8")),
    ("highest Unicode scalar", "\U0010ffff\n".encode("utf-8"), "\U0010ffff\n".encode("utf-8")),
]
RUNTIME_TEXT_FAILURES = [
    ("stray continuation", b"\x80\n"),
    ("overlong encoding", b"\xc0\xaf\n"),
    ("surrogate encoding", b"\xed\xa0\x80\n"),
    ("codepoint above Unicode", b"\xf4\x90\x80\x80\n"),
    ("overlong three-byte encoding", b"\xe0\x9f\xbf\n"),
    ("overlong four-byte encoding", b"\xf0\x8f\xbf\xbf\n"),
    ("invalid leading byte", b"\xf5\n"),
    ("truncated sequence", b"\xe2\x82\n"),
    ("line exceeds 63 bytes", b"a" * 64 + b"\n"),
]
RUNTIME_TEXT_VARIABLE_CASES = [
    ("overwrite and output variable", "旧值\n行简新值\r\n".encode("utf-8"), "行简新值\n".encode("utf-8")),
    ("maximum stored line", b"first\n" + b"x" * 63 + b"\n", b"x" * 63 + b"\n"),
    ("multiple growths with multibyte text", b"first\n" + ("𠮷" * 100 + "\r\n").encode("utf-8"), ("𠮷" * 100 + "\n").encode("utf-8")),
]
RUNTIME_TEXT_VARIABLE_FAILURES = [
    ("invalid UTF-8 during reassignment", b"old\n\xed\xa0\x80\n"),
]
RUNTIME_TEXT_CONCAT_CASES = [
    ("ASCII inputs", b"hello\nworld\n", b"helloworld\n"),
    ("Chinese and ASCII inputs", "行简\nSerica\n".encode("utf-8"), "行简Serica\n".encode("utf-8")),
    ("CRLF inputs", b"left\r\n-right\r\n", b"left-right\n"),
    ("empty values", b"\n\n", b"\n"),
    ("exact 63-byte result", b"x" * 60 + b"\nyyy\n", b"x" * 60 + b"yyy\n"),
    ("63-byte value plus empty", b"z" * 63 + b"\n\n", b"z" * 63 + b"\n"),
    ("64-byte result grows destination", b"x" * 63 + b"\ny\n", b"x" * 63 + b"y\n"),
    ("multiple growths with supplementary codepoints", ("𠮷" * 100 + "\n尾\n").encode("utf-8"), ("𠮷" * 100 + "尾\n").encode("utf-8")),
]
RUNTIME_TEXT_CONCAT_OUTPUT_CASES = RUNTIME_TEXT_CONCAT_CASES[:6]
RUNTIME_TEXT_CONCAT_LITERAL_CASES = [
    ("ASCII input with static prefix and constant suffix", b"hello\n", "前缀hello后缀\n".encode("utf-8")),
    ("Chinese input", "行简\n".encode("utf-8"), "前缀行简后缀\n".encode("utf-8")),
    ("empty input", b"\n", "前缀后缀\n".encode("utf-8")),
    ("exact 63-byte result", b"x" * 51 + b"\n", "前缀".encode("utf-8") + b"x" * 51 + "后缀\n".encode("utf-8")),
]
RUNTIME_TEXT_CONCAT_LITERAL_FAILURES = [
    ("static prefix causes overflow", b"x" * 58 + b"\n"),
    ("constant suffix causes overflow", b"x" * 52 + b"\n"),
]
RUNTIME_TEXT_CONCAT_READ_CASES = [
    ("readText on both sides", "行\n简\n".encode("utf-8"), "行尾首简\n".encode("utf-8")),
    ("empty and ASCII inputs", b"\nx\n", "尾首x\n".encode("utf-8")),
]
RUNTIME_TEXT_CONCAT_READ_FAILURES = [
    ("invalid UTF-8 in first input", b"\xff\nok\n"),
    ("invalid UTF-8 in second input", b"ok\n\xff\n"),
    ("first input exceeds 63 bytes", b"x" * 64 + b"\nok\n"),
    ("second input exceeds 63 bytes", b"ok\n" + b"x" * 64 + b"\n"),
]
RUNTIME_TEXT_CONCAT_READ_SLICE_CASES = [
    ("Chinese code-point slice", "A行简Z\n".encode("utf-8"), "行简尾\n".encode("utf-8")),
    ("supplementary scalar boundary", "😀甲乙丁\n".encode("utf-8"), "甲乙尾\n".encode("utf-8")),
]
RUNTIME_TEXT_CONCAT_READ_SLICE_FAILURES = [
    ("slice endpoint past input", b"A\n"),
    ("invalid UTF-8 input", b"\xed\xa0\x80\n"),
    ("input exceeds 63 bytes", b"x" * 64 + b"\n"),
]
RUNTIME_TEXT_ARRAY_SLICE_CASES = [
    ("dynamic text array CJK element", b"1\n", "行简\n".encode("utf-8")),
    ("dynamic text array supplementary scalar", b"2\n", "𠮷行\n".encode("utf-8")),
]
RUNTIME_TEXT_ARRAY_SLICE_FAILURES = [
    ("array index above length", b"3\n"),
    ("negative array index", b"-1\n"),
    ("slice endpoint past element", b"0\n"),
    ("invalid dynamic index", b"bad\n"),
]
RUNTIME_TEXT_ARRAY_SLICE_CONCAT_CASES = [
    ("CJK array element slice", b"1\n", "行简尾\n".encode("utf-8")),
    ("supplementary scalar array element slice", b"2\n", "𠮷行尾\n".encode("utf-8")),
]
RUNTIME_TEXT_ARRAY_SLICE_CONCAT_FAILURES = [
    ("array index above length", b"3\n"),
    ("negative array index", b"-1\n"),
    ("slice endpoint past element", b"0\n"),
    ("invalid dynamic index", b"bad\n"),
]
RUNTIME_TEXT_CONCAT_FAILURES = [
    ("result exceeds dynamic text limit", b"a" * (MAX_RUNTIME_TEXT_CAPACITY - 1) + b"\nx\n"),
]
RUNTIME_TEXT_CONCAT_OUTPUT_FAILURES = [
    ("64-byte result", b"a" * 63 + b"\nb\n"),
    ("sum above 63-byte limit", b"a" * 32 + b"\n" + b"b" * 32 + b"\n"),
    ("dynamic input above 63-byte limit", b"a" * 100 + b"\nb\n"),
]
RUNTIME_TEXT_COMPARISON_CASES = [
    ("equal Chinese values", "行简\n行简\n".encode("utf-8"), b"1\n"),
    ("different Chinese values", "甲\n乙\n".encode("utf-8"), b"0\n"),
    ("different lengths", b"\nlonger\n", b"0\n"),
    ("empty values", b"\n\n", b"1\n"),
    ("equal values after multiple growths", ("𠮷" * 100 + "\n" + "𠮷" * 100 + "\n").encode("utf-8"), b"1\n"),
]
RUNTIME_TEXT_STATIC_COMPARISON_CASES = [
    ("constant on the left matches", "行简\n".encode("utf-8"), b"1\n"),
    ("literal on the right differs", b"Serica\n", b"2\n"),
    ("empty input differs", b"\n", b"2\n"),
    ("maximum input differs", b"x" * 63 + b"\n", b"2\n"),
]
RUNTIME_TEXT_LENGTH_CASES = [
    ("ASCII", b"serica\n", b"6\n"),
    ("Chinese", "行简\n".encode("utf-8"), b"2\n"),
    ("supplementary scalar", "𠮷行\n".encode("utf-8"), b"2\n"),
    ("decomposed combining sequence", "e\u0301\n".encode("utf-8"), b"2\n"),
    ("precomposed scalar", "é\n".encode("utf-8"), b"1\n"),
    ("mixed-width text", "a𠮷中\n".encode("utf-8"), b"3\n"),
    ("empty", b"\n", b"0\n"),
    ("CRLF", "行简\r\n".encode("utf-8"), b"2\n"),
    ("63 ASCII scalars", b"x" * 63 + b"\n", b"63\n"),
]
RUNTIME_TEXT_NOT_EQUAL_CASES = [
    ("equal values", b"same\nsame\n", b"0\n"),
    ("different values", b"left\nright\n", b"1\n"),
    ("different lengths", b"\nlonger\n", b"1\n"),
]
RUNTIME_TEXT_SHORT_CIRCUIT_CASES = [
    ("equal values evaluates integer condition", b"same\nsame\n1\n", b"1\n"),
    ("unequal values skips integer input", b"same\nother\n", b"0\n"),
]
RUNTIME_TEXT_ORDERING_CASES = [
    ("ASCII less", b"a\nb\n", b"3\n"),
    ("ASCII greater", b"z\na\n", b"12\n"),
    ("equal", b"same\nsame\n", b"10\n"),
    ("prefix less", b"a\nab\n", b"3\n"),
    ("prefix greater", b"ab\na\n", b"12\n"),
    ("empty less", b"\ntext\n", b"3\n"),
    ("empty greater", b"text\n\n", b"12\n"),
    ("Chinese code-point order", "丁\n丂\n".encode("utf-8"), b"3\n"),
    ("four-byte code-point order", "😀\n😁\n".encode("utf-8"), b"3\n"),
    ("no Unicode normalization", "é\ne\u0301\n".encode("utf-8"), b"12\n"),
    ("long common prefix", ("x" * 100 + "a\n" + "x" * 100 + "b\n").encode("utf-8"), b"3\n"),
]
RUNTIME_TEXT_ORDER_EXPRESSION_CASES = [
    ("variable, concat, slice, array and direct input", "丁丂\nA\na\n丁丂\n".encode("utf-8"), b"47\n"),
    ("false slice and array, mixed static operand", "行简\nZ\nz\n行简\n".encode("utf-8"), b"17\n"),
]
RUNTIME_TEXT_SLICE_CASES = [
    ("ASCII scalars", b"abcd\n", b"bc\n"),
    ("Chinese scalars", "A行简Z\n".encode("utf-8"), "行简\n".encode("utf-8")),
    ("four-byte scalar boundary", "😀行简Z\n".encode("utf-8"), "行简\n".encode("utf-8")),
    ("maximum 63-byte line", b"x" * 63 + b"\n", b"x" * 2 + b"\n"),
]
RUNTIME_TEXT_READ_SLICE_CASES = [
    ("direct input code-point slice", "A行简Z\n".encode("utf-8"), "行简\n".encode("utf-8")),
]
RUNTIME_TEXT_READ_SLICE_FAILURES = [
    ("short input", b"A\n"),
    ("invalid UTF-8", b"\xff\n"),
    ("input exceeds 63-byte bound", b"x" * 64 + b"\n"),
]
RUNTIME_TEXT_ALIAS_SLICE_CASES = [
    ("ASCII alias update", b"abcd\n", b"bc\n"),
    ("multibyte alias update", "A行简Z\n".encode("utf-8"), "行简\n".encode("utf-8")),
]
RUNTIME_TEXT_SLICE_FAILURES = [("requested endpoint exceeds runtime text", b"ab\n")]
RUNTIME_TEXT_DYNAMIC_SLICE_CASES = [
    ("runtime ASCII indexes", b"abcd\n1\n3\n", b"bc\n"),
    ("runtime Chinese indexes", "A行简Z\n1\n3\n".encode("utf-8"), "行简\n".encode("utf-8")),
    ("runtime four-byte scalar indexes", "😀行简Z\n1\n3\n".encode("utf-8"), "行简\n".encode("utf-8")),
    ("runtime empty range", b"x\n1\n1\n", b"\n"),
    ("runtime maximum index", b"x" * 63 + b"\n62\n63\n", b"x\n"),
]
RUNTIME_TEXT_DYNAMIC_SLICE_FAILURES = [
    ("negative runtime start", b"abc\n-1\n2\n"),
    ("reversed runtime indexes", b"abc\n2\n1\n"),
    ("runtime end above limit", b"abc\n0\n64\n"),
    ("runtime endpoint past input", b"ab\n0\n3\n"),
    ("root output slice exceeds 63 bytes", ("𠮷" * 63 + "\n0\n63\n").encode("utf-8")),
]
RUNTIME_TEXT_CONCAT_SLICE_CASES = [
    ("slice crosses into static suffix", b"a\n", "𠮷x\n".encode("utf-8")),
    ("slice crosses dynamic/static boundary", "行简\n".encode("utf-8"), "简𠮷\n".encode("utf-8")),
    ("exact 63-byte concatenation", b"x" * 58 + b"\n", b"xx\n"),
    ("CRLF dynamic prefix", b"ab\r\n", "b𠮷\n".encode("utf-8")),
]
RUNTIME_TEXT_CONCAT_SLICE_FAILURES = [
    ("slice endpoint exceeds concatenated source", b"\n"),
    ("concatenation exceeds 63 bytes", b"x" * 59 + b"\n"),
    ("invalid UTF-8 prefix", b"\xed\xa0\x80\n"),
]
RUNTIME_TEXT_CONCAT_SLICE_LENGTH_CASES = [
    ("slice crosses into static suffix", b"a\n", b"2\n"),
    ("slice crosses dynamic/static boundary", "行简\n".encode("utf-8"), b"2\n"),
    ("exact 63-byte concatenation", b"x" * 58 + b"\n", b"2\n"),
    ("CRLF dynamic prefix", b"ab\r\n", b"2\n"),
]
RUNTIME_TEXT_CONCAT_SLICE_OPERAND_CASES = [
    ("63 four-byte scalars plus suffix", ("𠮷" * 100 + "\n0\n63\n").encode("utf-8"), ("𠮷" * 63 + "尾\n").encode("utf-8")),
    ("multibyte alias writeback", "A行简Z\n0\n4\n".encode("utf-8"), "A行简Z尾\n".encode("utf-8")),
]


def run(*args: str) -> subprocess.CompletedProcess[str]:
    # Inline runtime fixtures predate D-0063; write their least required entry capabilities.
    if len(args) > 1 and args[0] in {"check", "build"}:
        source = pathlib.Path(args[1])
        if source.suffix == ".serica" and source.is_file():
            contents = source.read_text(encoding="utf-8")
            if "程序=【" in contents and "能力=" not in contents:
                capabilities = ["stdout"]
                if "readInteger（" in contents or "readText（" in contents:
                    capabilities.append("stdin")
                if "arrayPush（" in contents or "arraySet（" in contents:
                    capabilities.append("array.write")
                declaration = "能力=array（" + "，".join(f"『{item}』" for item in capabilities) + "），"
                source.write_text(contents.replace("程序=【", "程序=【" + declaration, 1), encoding="utf-8")
    return subprocess.run(
        [str(SERICA), *args], cwd=ROOT, text=True, capture_output=True, check=False
    )


def run_unmodified(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(SERICA), *args], cwd=ROOT, text=True, capture_output=True, check=False
    )


def limit_file_size() -> None:
    resource.setrlimit(resource.RLIMIT_FSIZE, (0, 0))
    signal.signal(signal.SIGXFSZ, signal.SIG_IGN)


def verify_native_text_storage_artifact(path: pathlib.Path, **options: object) -> None:
    imports = set(options.pop("required_imports", ()))
    imports.update(("_mmap", "_munmap"))
    verify_native_console_artifact(path, required_imports=tuple(sorted(imports)), **options)


def main() -> int:
    with tempfile.TemporaryDirectory(prefix="serica-p1b-runtime-") as temporary:
        temp = pathlib.Path(temporary)
        for index, (label, source_text, expected) in enumerate(CASES):
            source = temp / f"case-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            if checked.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr)
                print(f"原生整数运行时静态检查失败：{label}", file=sys.stderr)
                return 1
            output = temp / f"case-{index}"
            built = run("build", str(source), "-o", str(output))
            if built.returncode != 0:
                sys.stderr.write(built.stdout + built.stderr)
                print(f"原生整数运行时构建失败：{label}", file=sys.stderr)
                return 1
            try:
                required_instructions = {
                    "variable addition and multiplication": (
                        "mul\tx0, x0, x1",
                        "adds\tx0, x0, x1",
                    ),
                    "variable subtraction and division": (
                        "subs\tx0, x0, x1",
                        "sdiv\tx0, x0, x1",
                    ),
                    "variable bitwise": ("eor\tx0, x0, x1",),
                    "negative remainder": (
                        "sdiv\tx2, x0, x1",
                        "msub\tx0, x2, x1, x0",
                    ),
                    "shift precedence": ("lsl\tx0, x0, x1",),
                    "arithmetic right shift": ("asr\tx0, x0, x1",),
                    "negative left shift": ("lsl\tx0, x0, x1",),
                    "variable remainder and shift": (
                        "sdiv\tx2, x0, x1",
                        "msub\tx0, x2, x1, x0",
                        "lsl\tx0, x0, x1",
                    ),
                    "nested condition and loop statements": (
                        "b.ge",
                        "b.gt",
                        "b\t",
                    ),
                    "nested loop break and continue": ("b\t",),
                }.get(label, ())
                verify_native_console_artifact(
                    output,
                    require_variable_storage="variable" in label or "negative variable" in label or "loop" in label,
                    required_instructions=required_instructions,
                    required_imports=("_mmap", "_munmap") if label == "dynamic integer array length" else (),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=ROOT, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"原生整数运行结果不符：{label}，期望 {expected!r}，"
                    f"退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1
            if expected.encode("utf-8") in output.read_bytes():
                print(f"产物意外包含预写输出文本：{label}", file=sys.stderr)
                return 1

        argument_examples = [
            (
                "argument count",
                "程序=【能力=array（『stdout』，『args』），输出=argumentCount（）】。\n",
                ["甲", "乙"],
                "3\n",
                (),
            ),
            (
                "argument text with constant index",
                "常量 下标=1。\n程序=【能力=array（『stdout』，『args』），输出=argumentText（下标）】。\n",
                ["甲乙", "丙"],
                "甲乙\n",
                (),
            ),
            (
                "argument text through declared function effect",
                "函数 首参数（）能力=array（『args』）=argumentText（1）。\n"
                "程序=【能力=array（『stdout』，『args』），输出=首参数（）】。\n",
                ["行简"],
                "行简\n",
                (),
            ),
            (
                "argument text with integer variable index",
                "变量 下标=1。\n程序=【能力=array（『stdout』，『args』），输出=argumentText（下标）】。\n",
                ["甲乙", "丙"],
                "甲乙\n",
                (),
            ),
            (
                "argument text stored in dynamic text variable",
                "变量 下标=2。\n变量 文本=argumentText（下标）。\n"
                "程序=【能力=array（『stdout』，『args』），输出=文本】。\n",
                ["甲乙", "存" * 100],
                "存" * 100 + "\n",
                ("_mmap", "_munmap"),
            ),
            (
                "argument text assigned over existing text variable",
                "变量 旧下标=1。\n变量 新下标=2。\n变量 文本=argumentText（旧下标）。\n赋值 文本=argumentText（新下标）。\n"
                "程序=【能力=array（『stdout』，『args』），输出=文本】。\n",
                ["甲乙", "覆盖"],
                "覆盖\n",
                ("_mmap", "_munmap"),
            ),
            (
                "argument text stored in dynamic text array",
                "变量 下标=1。\n变量 文本=array（argumentText（下标），『尾』）。\n"
                "程序=【能力=array（『stdout』，『args』），输出=arrayGet（文本，0）】。\n",
                ["甲乙", "数组参数"],
                "甲乙\n",
                ("_mmap", "_munmap"),
            ),
            (
                "argument text length without storage",
                "变量 下标=2。\n"
                "程序=【能力=array（『stdout』，『args』），输出=textLength（argumentText（下标））】。\n",
                ["甲乙", "中文🙂"],
                "3\n",
                (),
            ),
            (
                "argument text in root concatenation",
                "程序=【能力=array（『stdout』，『args』），输出=textConcat（argumentText（1），『尾』）】。\n",
                ["行简"],
                "行简尾\n",
                (),
            ),
            (
                "argument text as root substring source",
                "程序=【能力=array（『stdout』，『args』），输出=substring（argumentText（1），1，3）】。\n",
                ["甲乙丙"],
                "乙丙\n",
                (),
            ),
            (
                "argument text equality without storage",
                "程序=【能力=array（『stdout』，『args』），输出=equals（argumentText（1），『行简』）】。\n",
                ["行简"],
                "1\n",
                (),
            ),
            (
                "argument text success result payload",
                "变量 结果=成功（argumentText（1））。\n"
                "程序=【能力=array（『stdout』，『args』），输出=结果值（结果）】。\n",
                ["参数结果🙂"],
                "参数结果🙂\n",
                ("_mmap", "_munmap"),
            ),
            (
                "argument text result length",
                "变量 结果=成功（argumentText（1））。\n"
                "程序=【能力=array（『stdout』，『args』），输出=textLength（结果值（结果））】。\n",
                ["中文🙂"],
                "3\n",
                ("_mmap", "_munmap"),
            ),
        ]
        for index, (label, source_text, supplied, expected, required_imports) in enumerate(argument_examples):
            source = temp / f"arguments-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run_unmodified("check", str(source))
            output = temp / f"arguments-{index}"
            built = run_unmodified("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"命令行参数产物检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    output,
                    require_integer=label in {"argument count", "argument text length without storage", "argument text result length"},
                    required_imports=required_imports,
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output), *supplied], cwd=ROOT, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"命令行参数运行不符：{label}，期望 {expected!r}，"
                    f"实际 stdout={executed.stdout!r}，退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        missing_argument_source = temp / "arguments-missing.serica"
        missing_argument_source.write_text(
            "程序=【能力=array（『stdout』，『args』），输出=argumentText（1）】。\n",
            encoding="utf-8",
        )
        missing_argument_output = temp / "arguments-missing"
        built = run_unmodified(
            "build", str(missing_argument_source), "-o", str(missing_argument_output)
        )
        if built.returncode != 0:
            sys.stderr.write(built.stdout + built.stderr)
            print("缺少命令行实参的运行时候选产物无法构建", file=sys.stderr)
            return 1
        missing_argument = subprocess.run(
            [str(missing_argument_output)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if missing_argument.returncode != 2 or missing_argument.stdout or missing_argument.stderr:
            sys.stderr.write(missing_argument.stdout + missing_argument.stderr)
            print("缺少命令行实参时未静默退出 2", file=sys.stderr)
            return 1

        dynamic_argument_source = temp / "arguments-dynamic-edge.serica"
        dynamic_argument_source.write_text(
            "变量 下标=readInteger（）。\n"
            "变量 文本=argumentText（下标）。\n"
            "程序=【能力=array（『stdout』，『stdin』，『args』），输出=文本】。\n",
            encoding="utf-8",
        )
        dynamic_argument_output = temp / "arguments-dynamic-edge"
        built = run_unmodified(
            "build", str(dynamic_argument_source), "-o", str(dynamic_argument_output)
        )
        if built.returncode != 0:
            sys.stderr.write(built.stdout + built.stderr)
            print("动态命令行下标产物无法构建", file=sys.stderr)
            return 1
        for supplied, stdin_text, expected_code, expected_output in [
            (["甲乙", "丙"], "2\n", 0, "丙\n"),
            (["甲乙"], "-1\n", 2, ""),
            (["甲乙"], "9\n", 2, ""),
            (["a" * 4095], "1\n", 0, "a" * 4095 + "\n"),
            (["a" * 4096], "1\n", 3, ""),
        ]:
            executed = subprocess.run(
                [str(dynamic_argument_output), *supplied],
                cwd=ROOT,
                input=stdin_text,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != expected_code or executed.stdout != expected_output or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"动态命令行下标边界不符：输入参数 {supplied!r}、stdin={stdin_text!r}，"
                    f"期望退出 {expected_code} 和 stdout={expected_output!r}，"
                    f"实际退出 {executed.returncode} 和 stdout={executed.stdout!r}",
                    file=sys.stderr,
                )
                return 1

        invalid_argument = subprocess.run(
            [str(dynamic_argument_output).encode(), b"\xff"],
            cwd=ROOT,
            input=b"1\n",
            capture_output=True,
            check=False,
        )
        if invalid_argument.returncode == 0 or invalid_argument.stdout or invalid_argument.stderr:
            sys.stderr.buffer.write(invalid_argument.stdout + invalid_argument.stderr)
            print("无效 UTF-8 命令行参数未保持零 stdout 并非零退出", file=sys.stderr)
            return 1

        maximum_file_bytes = MAX_RUNTIME_TEXT_CAPACITY - 1
        file_reads = [
            ("ASCII file", "ascii.txt", b"hello\n", b"hello\n\n", 0),
            ("Chinese filename and UTF-8 file", "中文.txt", "行简\n".encode("utf-8"), "行简\n\n".encode("utf-8"), 0),
            ("empty file", "empty.txt", b"", b"\n", 0),
            (
                "maximum legal file",
                "maximum.txt",
                b"a" * maximum_file_bytes,
                b"a" * maximum_file_bytes + b"\n",
                0,
            ),
            ("missing file", "missing.txt", None, b"", 1),
            ("invalid UTF-8 file", "invalid.txt", b"\xed\xa0\x80", b"", 1),
            (
                "oversize file",
                "oversize.txt",
                b"x" * MAX_RUNTIME_TEXT_CAPACITY,
                b"",
                1,
            ),
        ]
        for index, (label, filename, contents, expected, expected_code) in enumerate(file_reads):
            path = temp / filename
            if contents is not None:
                path.write_bytes(contents)
            source = temp / f"file-read-{index}.serica"
            source.write_text(
                f"程序=【能力=array（『stdout』，『filesystem.read』），输出=readFile（『{filename}』）】。\n",
                encoding="utf-8",
            )
            checked = run_unmodified("check", str(source))
            output = temp / f"file-read-{index}"
            built = run_unmodified("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"静态文件读取产物检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    output,
                    require_integer=False,
                    required_imports=("_open", "_close", "_read", "_mmap", "_munmap"),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=temp, capture_output=True, check=False, timeout=15
            )
            if (
                executed.returncode != expected_code
                or executed.stdout != expected
                or executed.stderr
            ):
                sys.stderr.buffer.write(executed.stdout[:256] + executed.stderr[:256])
                print(
                    f"静态文件读取运行不符：{label}，期望 {expected_code}/{len(expected)} 字节，"
                    f"实际 {executed.returncode}/{len(executed.stdout)} 字节",
                    file=sys.stderr,
                )
                return 1

        dynamic_read_path = temp / "动态读取.txt"
        dynamic_read_path.write_text("动态路径🙂\n", encoding="utf-8")
        dynamic_read_source = temp / "file-read-dynamic-path.serica"
        dynamic_read_source.write_text(
            "程序=【能力=array（『stdout』，『args』，『filesystem.read』），输出=readFile（argumentText（1））】。\n",
            encoding="utf-8",
        )
        dynamic_read_output = temp / "file-read-dynamic-path"
        checked = run_unmodified("check", str(dynamic_read_source))
        built = run_unmodified("build", str(dynamic_read_source), "-o", str(dynamic_read_output))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态路径文件读取无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                dynamic_read_output,
                require_integer=False,
                require_runtime_input=True,
                required_imports=("_open", "_close", "_read", "_mmap", "_munmap"),
            )
        except RuntimeError as failure:
            print(f"动态路径文件读取：{failure}", file=sys.stderr)
            return 1
        executed = subprocess.run(
            [str(dynamic_read_output), str(dynamic_read_path)],
            cwd=temp,
            capture_output=True,
            check=False,
            timeout=15,
        )
        if executed.returncode != 0 or executed.stdout != "动态路径🙂\n\n".encode("utf-8") or executed.stderr:
            sys.stderr.buffer.write(executed.stdout + executed.stderr)
            print("动态路径文件读取运行不符", file=sys.stderr)
            return 1

        dynamic_read_rejections = [
            ("missing path argument", [], 2),
            ("maximum path rejected by open", ["a" * 4095], 1),
            ("oversize path argument", ["a" * 4096], 3),
        ]
        dynamic_file_path_failure_count = len(dynamic_read_rejections)
        for label, supplied, expected_code in dynamic_read_rejections:
            rejected_run = subprocess.run(
                [str(dynamic_read_output), *supplied],
                cwd=temp,
                capture_output=True,
                check=False,
                timeout=5,
            )
            if rejected_run.returncode != expected_code or rejected_run.stdout or rejected_run.stderr:
                sys.stderr.buffer.write(rejected_run.stdout[:256] + rejected_run.stderr[:256])
                print(
                    f"动态读取路径 {label} 不符：期望退出 {expected_code}，"
                    f"实际 {rejected_run.returncode}/{len(rejected_run.stdout)} 字节",
                    file=sys.stderr,
                )
                return 1
        invalid_read_path = subprocess.run(
            [str(dynamic_read_output).encode(), b"\xff"],
            cwd=temp,
            capture_output=True,
            check=False,
            timeout=5,
        )
        dynamic_file_path_failure_count += 1
        if invalid_read_path.returncode == 0 or invalid_read_path.stdout or invalid_read_path.stderr:
            sys.stderr.buffer.write(invalid_read_path.stdout[:256] + invalid_read_path.stderr[:256])
            print("动态读取路径未拒绝无效 UTF-8", file=sys.stderr)
            return 1

        file_function_source = temp / "file-read-function.serica"
        file_function_source.write_text(
            "函数 读取（）能力=array（『filesystem.read』）=readFile（『function.txt』）。\n"
            "程序=【能力=array（『stdout』，『filesystem.read』），输出=读取（）】。\n",
            encoding="utf-8",
        )
        (temp / "function.txt").write_text("函数读取\n", encoding="utf-8")
        file_function_output = temp / "file-read-function"
        checked = run_unmodified("check", str(file_function_source))
        built = run_unmodified("build", str(file_function_source), "-o", str(file_function_output))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("带 filesystem.read 效果的函数文件读取无法构建", file=sys.stderr)
            return 1
        executed = subprocess.run(
            [str(file_function_output)], cwd=temp, capture_output=True, check=False
        )
        if executed.returncode != 0 or executed.stdout != "函数读取\n\n".encode("utf-8") or executed.stderr:
            sys.stderr.buffer.write(executed.stdout + executed.stderr)
            print("带 filesystem.read 效果的函数文件读取运行不符", file=sys.stderr)
            return 1

        (temp / "variable-a.txt").write_text("甲乙", encoding="utf-8")
        (temp / "variable-b.txt").write_text("丙丁", encoding="utf-8")
        file_variable_cases = [
            (
                "file text variable output",
                "变量 文本=readFile（『variable-a.txt』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=文本】。\n",
                "甲乙\n".encode("utf-8"),
                False,
            ),
            (
                "file text variable Unicode scalar length",
                "变量 文本=readFile（『variable-a.txt』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=textLength（文本）】。\n",
                b"2\n",
                True,
            ),
            (
                "file text variable reassignment unmaps previous contents",
                "变量 文本=readFile（『variable-a.txt』）。\n"
                "赋值 文本=readFile（『variable-b.txt』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=文本】。\n",
                "丙丁\n".encode("utf-8"),
                False,
            ),
            (
                "file text variable concatenation",
                "变量 文本=readFile（『variable-a.txt』）。\n"
                "赋值 文本=textConcat（文本，『！』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=文本】。\n",
                "甲乙！\n".encode("utf-8"),
                False,
            ),
        ]
        for index, (label, source_text, expected, integer_output) in enumerate(file_variable_cases):
            source = temp / f"file-variable-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run_unmodified("check", str(source))
            output = temp / f"file-variable-{index}"
            built = run_unmodified("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"文件文本变量产物检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    output,
                    require_integer=integer_output,
                    require_variable_storage=False,
                    required_instructions=("str\tx21, [x19]", "str\tx28, [x19, #0x8]"),
                    required_imports=("_open", "_close", "_read", "_mmap", "_munmap"),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=temp, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"文件文本变量运行不符：{label}", file=sys.stderr)
                return 1

        file_variable_failures = [
            ("invalid UTF-8", "variable-invalid.txt", b"\xed\xa0\x80"),
            ("oversize", "variable-oversize.txt", b"x" * MAX_RUNTIME_TEXT_CAPACITY),
        ]
        for index, (label, filename, contents) in enumerate(file_variable_failures):
            (temp / filename).write_bytes(contents)
            source = temp / f"file-variable-failure-{index}.serica"
            source.write_text(
                f"变量 文本=readFile（『{filename}』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=文本】。\n",
                encoding="utf-8",
            )
            output = temp / f"file-variable-failure-{index}"
            built = run_unmodified("build", str(source), "-o", str(output))
            if built.returncode != 0:
                sys.stderr.write(built.stdout + built.stderr)
                print(f"文件文本变量失败候选无法构建：{label}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=temp, capture_output=True, check=False, timeout=15
            )
            if executed.returncode == 0 or executed.stdout or executed.stderr:
                sys.stderr.buffer.write(executed.stdout[:256] + executed.stderr[:256])
                print(f"文件文本变量 {label} 未无输出失败", file=sys.stderr)
                return 1

        file_write_cases = [
            (
                "static UTF-8 overwrite and byte count",
                {"write-target.txt": "旧内容".encode("utf-8")},
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『write-target.txt』，『甲乙！』）】。\n",
                b"9\n",
                "write-target.txt",
                "甲乙！".encode("utf-8"),
                True,
                ("_creat", "_close"),
            ),
            (
                "empty text truncates existing file",
                {"empty-write.txt": b"old contents"},
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『empty-write.txt』，『』）】。\n",
                b"0\n",
                "empty-write.txt",
                b"",
                True,
                ("_creat", "_close"),
            ),
            (
                "supplementary-plane UTF-8 byte count",
                {},
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『supplementary.txt』，『𠮷🙂』）】。\n",
                b"8\n",
                "supplementary.txt",
                "𠮷🙂".encode("utf-8"),
                True,
                ("_creat", "_close"),
            ),
            (
                "dynamic text read from one file and written to another",
                {"write-source.txt": "行简！".encode("utf-8")},
                "变量 文本=readFile（『write-source.txt』）。\n"
                "变量 字节数=writeFile（『write-result.txt』，文本）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』，『filesystem.write』），"
                "输出=字节数】。\n",
                b"9\n",
                "write-result.txt",
                "行简！".encode("utf-8"),
                True,
                ("_open", "_read", "_mmap", "_munmap", "_creat", "_close"),
            ),
            (
                "writeFile through declared function effect",
                {},
                "函数 写入（）能力=array（『filesystem.write』）="
                "writeFile（『function-write.txt』，『函数写入』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.write』），输出=写入（）】。\n",
                b"12\n",
                "function-write.txt",
                "函数写入".encode("utf-8"),
                True,
                ("_creat", "_close"),
            ),
        ]
        for index, (
            label,
            initial_files,
            source_text,
            expected_stdout,
            written_name,
            expected_contents,
            integer_output,
            required_imports,
        ) in enumerate(file_write_cases):
            for filename, contents in initial_files.items():
                (temp / filename).write_bytes(contents)
            source = temp / f"file-write-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run_unmodified("check", str(source))
            output = temp / f"file-write-{index}"
            built = run_unmodified("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"文件写入产物检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    output,
                    require_integer=integer_output,
                    required_imports=required_imports,
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=temp, capture_output=True, check=False, timeout=5
            )
            actual_contents = (temp / written_name).read_bytes()
            if (
                executed.returncode != 0
                or executed.stdout != expected_stdout
                or executed.stderr
                or actual_contents != expected_contents
            ):
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(
                    f"文件写入运行不符：{label}，实际 stdout={executed.stdout!r}，"
                    f"文件={actual_contents[:64]!r}，退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        dynamic_write_path = temp / "动态写入.txt"
        dynamic_write_source = temp / "file-write-dynamic-path.serica"
        dynamic_write_source.write_text(
            "变量 字节=writeFile（argumentText（1），argumentText（2））。\n"
            "程序=【能力=array（『stdout』，『args』，『filesystem.write』），输出=字节】。\n",
            encoding="utf-8",
        )
        dynamic_write_output = temp / "file-write-dynamic-path"
        checked = run_unmodified("check", str(dynamic_write_source))
        built = run_unmodified("build", str(dynamic_write_source), "-o", str(dynamic_write_output))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态路径文件写入无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                dynamic_write_output,
                require_integer=True,
                require_variable_storage=True,
                required_imports=("_creat", "_close"),
            )
        except RuntimeError as failure:
            print(f"动态路径文件写入：{failure}", file=sys.stderr)
            return 1
        executed = subprocess.run(
            [str(dynamic_write_output), str(dynamic_write_path), "行简"],
            cwd=temp,
            capture_output=True,
            check=False,
            timeout=5,
        )
        if (
            executed.returncode != 0
            or executed.stdout != b"6\n"
            or executed.stderr
            or dynamic_write_path.read_bytes() != "行简".encode("utf-8")
        ):
            sys.stderr.buffer.write(executed.stdout + executed.stderr)
            print("动态路径文件写入运行不符", file=sys.stderr)
            return 1

        dynamic_write_rejections = [
            ("missing path and content arguments", [], 2),
            ("oversize path argument", ["a" * 4096, "x"], 3),
        ]
        dynamic_file_path_failure_count += len(dynamic_write_rejections)
        for label, supplied, expected_code in dynamic_write_rejections:
            rejected_run = subprocess.run(
                [str(dynamic_write_output), *supplied],
                cwd=temp,
                capture_output=True,
                check=False,
                timeout=5,
            )
            if rejected_run.returncode != expected_code or rejected_run.stdout or rejected_run.stderr:
                sys.stderr.buffer.write(rejected_run.stdout[:256] + rejected_run.stderr[:256])
                print(
                    f"动态写入路径 {label} 不符：期望退出 {expected_code}，"
                    f"实际 {rejected_run.returncode}/{len(rejected_run.stdout)} 字节",
                    file=sys.stderr,
                )
                return 1
        invalid_write_path = subprocess.run(
            [str(dynamic_write_output).encode(), b"\xff", b"x"],
            cwd=temp,
            capture_output=True,
            check=False,
            timeout=5,
        )
        dynamic_file_path_failure_count += 1
        if invalid_write_path.returncode == 0 or invalid_write_path.stdout or invalid_write_path.stderr:
            sys.stderr.buffer.write(invalid_write_path.stdout[:256] + invalid_write_path.stderr[:256])
            print("动态写入路径未拒绝无效 UTF-8 路径", file=sys.stderr)
            return 1
        invalid_write_target = temp / "invalid-dynamic-content-target.txt"
        invalid_write_target.write_bytes(b"preserved")
        invalid_write_content = subprocess.run(
            [str(dynamic_write_output).encode(), str(invalid_write_target).encode(), b"\xff"],
            cwd=temp,
            capture_output=True,
            check=False,
            timeout=5,
        )
        dynamic_file_path_failure_count += 1
        if (
            invalid_write_content.returncode == 0
            or invalid_write_content.stdout
            or invalid_write_content.stderr
            or invalid_write_target.read_bytes() != b"preserved"
        ):
            sys.stderr.buffer.write(invalid_write_content.stdout[:256] + invalid_write_content.stderr[:256])
            print("动态写入内容未在截断目标前拒绝无效 UTF-8", file=sys.stderr)
            return 1

        file_write_failures = [
            (
                "missing parent directory",
                False,
                None,
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『missing/parent/result.txt』，『x』）】。\n",
                21,
                None,
                None,
                ("_creat", "_close"),
            ),
            (
                "directory target",
                True,
                None,
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『write-directory』，『x』）】。\n",
                21,
                None,
                None,
                ("_creat", "_close"),
            ),
            (
                "invalid source leaves destination untouched",
                False,
                {"invalid-write-source.txt": b"\xed\xa0\x80"},
                "变量 文本=readFile（『invalid-write-source.txt』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』，『filesystem.write』），"
                "输出=writeFile（『preserved-target.txt』，文本）】。\n",
                1,
                "preserved-target.txt",
                b"unchanged",
                ("_open", "_read", "_mmap", "_munmap", "_creat", "_close"),
            ),
        ]
        for index, (
            label,
            make_directory,
            initial_files,
            source_text,
            expected_code,
            preserved_name,
            preserved_contents,
            required_imports,
        ) in enumerate(file_write_failures):
            if make_directory:
                (temp / "write-directory").mkdir()
            for filename, contents in (initial_files or {}).items():
                (temp / filename).write_bytes(contents)
            if preserved_name is not None:
                (temp / preserved_name).write_bytes(preserved_contents or b"")
            source = temp / f"file-write-failure-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run_unmodified("check", str(source))
            output = temp / f"file-write-failure-{index}"
            built = run_unmodified("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"文件写入失败候选检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(output, required_imports=required_imports)
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=temp, capture_output=True, check=False, timeout=5
            )
            preserved = None if preserved_name is None else (temp / preserved_name).read_bytes()
            if (
                executed.returncode != expected_code
                or executed.stdout
                or executed.stderr
                or preserved is not None and preserved != preserved_contents
            ):
                sys.stderr.buffer.write(executed.stdout[:256] + executed.stderr[:256])
                print(
                    f"文件写入失败路径不符：{label}，期望 {expected_code}，"
                    f"实际 {executed.returncode}，目标={preserved!r}",
                    file=sys.stderr,
                )
                return 1

        write_failure_source = temp / "file-write-write-failure.serica"
        write_failure_source.write_text(
            "程序=【能力=array（『stdout』，『filesystem.write』），"
            "输出=writeFile（『limited-write.txt』，『abc』）】。\n",
            encoding="utf-8",
        )
        write_failure_output = temp / "file-write-write-failure"
        checked = run_unmodified("check", str(write_failure_source))
        built = run_unmodified(
            "build", str(write_failure_source), "-o", str(write_failure_output)
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("文件写入错误候选检查或构建失败", file=sys.stderr)
            return 1
        executed = subprocess.run(
            [str(write_failure_output)],
            cwd=temp,
            capture_output=True,
            check=False,
            timeout=5,
            preexec_fn=limit_file_size,
        )
        partial = (temp / "limited-write.txt").read_bytes()
        if executed.returncode != 22 or executed.stdout or executed.stderr or partial:
            sys.stderr.buffer.write(executed.stdout[:256] + executed.stderr[:256])
            print(
                f"文件写入无进展路径不符：退出码 {executed.returncode}，目标={partial!r}",
                file=sys.stderr,
            )
            return 1

        for index, (label, source_text, expected) in enumerate(TEXT_RESULT_CASES):
            source = temp / f"text-result-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            output = temp / f"text-result-{index}"
            built = run("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"文本结果产物检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    output,
                    require_integer=expected == "0\n",
                    require_variable_storage=True,
                    required_imports=(),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)], cwd=ROOT, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"文本结果运行不符：{label}，期望 {expected!r}，"
                    f"实际 stdout={executed.stdout!r}，退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, (label, source_text, supplied, expected, expected_code) in enumerate(DYNAMIC_TEXT_RESULT_CASES):
            source = temp / f"dynamic-text-result-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            output = temp / f"dynamic-text-result-{index}"
            built = run("build", str(source), "-o", str(output))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"运行时文本结果产物检查或构建失败：{label}", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    output,
                    require_integer=False,
                    require_variable_storage=True,
                    require_runtime_input=True,
                    required_imports=("_mmap", "_munmap"),
                )
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(output)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if (
                executed.returncode != expected_code
                or executed.stdout != expected
                or executed.stderr
            ):
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"运行时文本结果不符：{label}，期望 {expected!r}/{expected_code}，"
                    f"实际 stdout={executed.stdout!r}，退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        project = temp / "variable-module"
        project.mkdir()
        (project / "base.serica").write_text("常量 起点=80。\n", encoding="utf-8")
        entry = project / "main.serica"
        entry.write_text(
            "导入 『base.serica』。\n变量 数值=起点。\n赋值 数值=84。\n程序=【输出=数值】。\n",
            encoding="utf-8",
        )
        module_artifact = temp / "variable-module-output"
        checked = run("check", str(entry))
        built = run("build", str(entry), "-o", str(module_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("静态模块常量与运行时变量组合失败", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(module_artifact, require_variable_storage=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        module_run = subprocess.run(
            [str(module_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if module_run.returncode != 0 or module_run.stdout != "84\n" or module_run.stderr:
            sys.stderr.write(module_run.stdout + module_run.stderr)
            print("静态模块常量未初始化运行时变量", file=sys.stderr)
            return 1

        max_variables_source = temp / "max-variables.serica"
        max_variables_source.write_text(
            "".join(f"变量 数值{index}=0。\n" for index in range(502))
            + "程序=【输出=数值501】。\n",
            encoding="utf-8",
        )
        max_variables_artifact = temp / "max-variables"
        checked = run("check", str(max_variables_source))
        built = run("build", str(max_variables_source), "-o", str(max_variables_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("502 个整数栈槽上限内的程序无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(max_variables_artifact)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        max_variables_run = subprocess.run(
            [str(max_variables_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if max_variables_run.returncode != 0 or max_variables_run.stdout != "0\n" or max_variables_run.stderr:
            sys.stderr.write(max_variables_run.stdout + max_variables_run.stderr)
            print("502 个整数栈槽的最高偏移读取失败", file=sys.stderr)
            return 1

        string_source = ROOT / "examples" / "p1b-runtime-string.serica"
        string_artifact = temp / "utf8-string"
        built = run("build", str(string_source), "-o", str(string_artifact))
        if built.returncode != 0:
            sys.stderr.write(built.stdout + built.stderr)
            print("UTF-8 字符串输出构建失败", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(string_artifact, require_integer=False)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        string_run = subprocess.run(
            [str(string_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if string_run.returncode != 0 or string_run.stdout != "行简字符串输出\n" or string_run.stderr:
            sys.stderr.write(string_run.stdout + string_run.stderr)
            print("原生 UTF-8 字符串输出结果不符", file=sys.stderr)
            return 1

        input_source = temp / "runtime-integer-input.serica"
        input_source.write_text("程序=【输出=readInteger（）】。\n", encoding="utf-8")
        input_artifact = temp / "runtime-integer-input"
        checked = run("check", str(input_source))
        built = run("build", str(input_source), "-o", str(input_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时整数输入子门无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                input_artifact,
                require_runtime_input=True,
                required_instructions=("ldrb\tw25, [x21]", "mul\tx23, x23, x26"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1

        for supplied, expected in [
            ("42\n", "42\n"),
            ("-17\r\n", "-17\n"),
            ("1\n2\n", "1\n"),
            ("9223372036854775807", "9223372036854775807\n"),
            ("-9223372036854775808\n", "-9223372036854775808\n"),
            ("-0\n", "0\n"),
        ]:
            executed = subprocess.run(
                [str(input_artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"运行时整数输入结果不符：{supplied!r}，期望 {expected!r}，"
                    f"实际 stdout={executed.stdout!r}，退出码={executed.returncode}",
                    file=sys.stderr,
                )
                return 1
        for supplied in [
            "",
            "\n",
            "-\n",
            "+1\n",
            "1x\n",
            "1" * 64 + "\n",
            "9223372036854775808\n",
            "-9223372036854775809\n",
        ]:
            executed = subprocess.run(
                [str(input_artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode == 0 or executed.stdout or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(f"非法运行时整数输入未以非零退出：{supplied!r}", file=sys.stderr)
                return 1

        function_source = temp / "runtime-pure-function.serica"
        function_source.write_text(
            "函数 倍增（数值）=数值*2。\n"
            "函数 三倍（数值）=倍增（数值）+数值。\n"
            "变量 输入=readInteger（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=三倍（输入）】。\n",
            encoding="utf-8",
        )
        function_artifact = temp / "runtime-pure-function"
        checked = run_unmodified("check", str(function_source))
        built = run_unmodified("build", str(function_source), "-o", str(function_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("嵌套纯函数的运行时调用无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(function_artifact, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        function_run = subprocess.run(
            [str(function_artifact)], cwd=ROOT, input="14\n", text=True, capture_output=True, check=False
        )
        if function_run.returncode != 0 or function_run.stdout != "42\n" or function_run.stderr:
            sys.stderr.write(function_run.stdout + function_run.stderr)
            print("嵌套纯函数的运行时结果不符", file=sys.stderr)
            return 1

        compound_function_source = temp / "runtime-compound-function.serica"
        compound_function_source.write_text(
            "函数 加一（数值）=数值+1。\n"
            "函数 双倍（数值）=数值+数值。\n"
            "变量 输入=readInteger（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=双倍（加一（输入））】。\n",
            encoding="utf-8",
        )
        compound_function_artifact = temp / "runtime-compound-function"
        checked = run_unmodified("check", str(compound_function_source))
        built = run_unmodified(
            "build",
            str(compound_function_source),
            "-o",
            str(compound_function_artifact),
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("带嵌套函数调用的复合实参无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                compound_function_artifact, require_runtime_input=True
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        compound_function_run = subprocess.run(
            [str(compound_function_artifact)],
            cwd=ROOT,
            input="20\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            compound_function_run.returncode != 0
            or compound_function_run.stdout != "42\n"
            or compound_function_run.stderr
        ):
            sys.stderr.write(compound_function_run.stdout + compound_function_run.stderr)
            print("带嵌套函数调用的复合实参结果不符", file=sys.stderr)
            return 1

        text_function_source = temp / "runtime-text-function.serica"
        text_function_source.write_text(
            "函数 加后缀（文本）=textConcat（文本，『！』）。\n"
            "变量 输入=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=加后缀（输入）】。\n",
            encoding="utf-8",
        )
        text_function_artifact = temp / "runtime-text-function"
        checked = run_unmodified("check", str(text_function_source))
        built = run_unmodified(
            "build", str(text_function_source), "-o", str(text_function_artifact)
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时文本参数纯函数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                text_function_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_imports=("_mmap", "_munmap"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        text_function_run = subprocess.run(
            [str(text_function_artifact)],
            cwd=ROOT,
            input="行简\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            text_function_run.returncode != 0
            or text_function_run.stdout != "行简！\n"
            or text_function_run.stderr
        ):
            sys.stderr.write(text_function_run.stdout + text_function_run.stderr)
            print("运行时文本参数纯函数结果不符", file=sys.stderr)
            return 1

        effect_function_source = temp / "runtime-effect-function.serica"
        effect_function_source.write_text(
            "函数 读取（）能力=array（『stdin』）=readInteger（）。\n"
            "函数 三倍输入（）能力=array（『stdin』）=读取（）*3。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=三倍输入（）】。\n",
            encoding="utf-8",
        )
        effect_function_artifact = temp / "runtime-effect-function"
        checked = run_unmodified("check", str(effect_function_source))
        built = run_unmodified(
            "build", str(effect_function_source), "-o", str(effect_function_artifact)
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("显式 stdin 效果的嵌套函数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                effect_function_artifact, require_runtime_input=True
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        effect_function_run = subprocess.run(
            [str(effect_function_artifact)],
            cwd=ROOT,
            input="14\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            effect_function_run.returncode != 0
            or effect_function_run.stdout != "42\n"
            or effect_function_run.stderr
        ):
            sys.stderr.write(effect_function_run.stdout + effect_function_run.stderr)
            print("显式 stdin 效果的嵌套函数结果不符", file=sys.stderr)
            return 1

        effect_block_direct_source = temp / "runtime-effect-block-direct.serica"
        effect_block_direct_source.write_text(
            "函数 翻倍（次数）能力=array（『stdin』）【\n"
            "    变量 合计=0。\n"
            "    变量 索引=0。\n"
            "    凡（索引<次数）【赋值 合计=合计+readInteger（）。赋值 索引=索引+1。】。\n"
            "    返回 合计。\n"
            "】。\n"
            "变量 输入=readInteger（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=翻倍（输入）】。\n",
            encoding="utf-8",
        )
        effect_block_direct_artifact = temp / "runtime-effect-block-direct"
        checked = run_unmodified("check", str(effect_block_direct_source))
        built = run_unmodified(
            "build", str(effect_block_direct_source), "-o", str(effect_block_direct_artifact)
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时循环中的直接 stdin 效果无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                effect_block_direct_artifact,
                require_runtime_input=True,
                required_instructions=("bl\t", "ret"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        effect_block_direct_run = subprocess.run(
            [str(effect_block_direct_artifact)],
            cwd=ROOT,
            input="3\n1\n2\n3\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            effect_block_direct_run.returncode != 0
            or effect_block_direct_run.stdout != "6\n"
            or effect_block_direct_run.stderr
        ):
            sys.stderr.write(effect_block_direct_run.stdout + effect_block_direct_run.stderr)
            print("运行时循环中的直接 stdin 效果结果不符", file=sys.stderr)
            return 1

        effect_block_loop_source = temp / "runtime-effect-block-loop.serica"
        effect_block_loop_source.write_text(
            "函数 读取倍数（）能力=array（『stdin』）【返回 readInteger（）*2。】。\n"
            "函数 四次（次数）能力=array（『stdin』）【\n"
            "    变量 合计=0。\n"
            "    变量 索引=0。\n"
            "    凡（索引<次数）【赋值 合计=合计+读取倍数（）。赋值 索引=索引+1。】。\n"
            "    返回 合计+次数。\n"
            "】。\n"
            "变量 输入=readInteger（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=四次（输入）】。\n",
            encoding="utf-8",
        )
        effect_block_loop_artifact = temp / "runtime-effect-block-loop"
        checked = run_unmodified("check", str(effect_block_loop_source))
        built = run_unmodified(
            "build", str(effect_block_loop_source), "-o", str(effect_block_loop_artifact)
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时循环中的显式 stdin 效果函数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                effect_block_loop_artifact,
                require_runtime_input=True,
                required_instructions=("bl\t", "ret"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        effect_block_loop_run = subprocess.run(
            [str(effect_block_loop_artifact)],
            cwd=ROOT,
            input="3\n1\n2\n3\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            effect_block_loop_run.returncode != 0
            or effect_block_loop_run.stdout != "15\n"
            or effect_block_loop_run.stderr
        ):
            sys.stderr.write(effect_block_loop_run.stdout + effect_block_loop_run.stderr)
            print("运行时循环中的显式 stdin 效果函数结果不符", file=sys.stderr)
            return 1

        effect_missing_capability_source = temp / "runtime-effect-block-missing-capability.serica"
        effect_missing_capability_source.write_text(
            "函数 读取倍数（）能力=array（『stdin』）【返回 readInteger（）*2。】。\n"
            "函数 四次（次数）【\n"
            "    变量 合计=0。\n"
            "    变量 索引=0。\n"
            "    凡（索引<次数）【赋值 合计=合计+读取倍数（）。赋值 索引=索引+1。】。\n"
            "    返回 合计+次数。\n"
            "】。\n"
            "变量 输入=readInteger（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=四次（输入）】。\n",
            encoding="utf-8",
        )
        checked = run_unmodified("check", str(effect_missing_capability_source))
        if checked.returncode == 0 or "S-4106" not in (checked.stdout + checked.stderr):
            sys.stderr.write(checked.stdout + checked.stderr)
            print("运行时循环中的隐藏效果应被 S-4106 拒绝", file=sys.stderr)
            return 1

        effect_text_function_source = temp / "runtime-effect-text-function.serica"
        effect_text_function_source.write_text(
            "函数 读取文本（）能力=array（『stdin』）=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=读取文本（）】。\n",
            encoding="utf-8",
        )
        effect_text_function_artifact = temp / "runtime-effect-text-function"
        checked = run_unmodified("check", str(effect_text_function_source))
        built = run_unmodified(
            "build",
            str(effect_text_function_source),
            "-o",
            str(effect_text_function_artifact),
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("显式 stdin 效果的文本函数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                effect_text_function_artifact,
                require_integer=False,
                require_runtime_input=True,
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        effect_text_function_run = subprocess.run(
            [str(effect_text_function_artifact)],
            cwd=ROOT,
            input="行简\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            effect_text_function_run.returncode != 0
            or effect_text_function_run.stdout != "行简\n"
            or effect_text_function_run.stderr
        ):
            sys.stderr.write(effect_text_function_run.stdout + effect_text_function_run.stderr)
            print("显式 stdin 效果的文本函数结果不符", file=sys.stderr)
            return 1

        array_effect_function_source = temp / "runtime-array-effect-function.serica"
        array_effect_function_source.write_text(
            "函数 替换（数组）能力=array（『array.write』）=arraySet（数组，0，7）。\n"
            "函数 追加（数组）能力=array（『array.write』）=arrayPush（数组，42）。\n"
            "函数 写入（数组）能力=array（『array.write』）=追加（数组）。\n"
            "变量 数组=array（7）。\n"
            "赋值 数组=替换（数组）。\n"
            "赋值 数组=写入（数组）。\n"
            "程序=【能力=array（『stdout』，『array.write』），"
            "输出=arrayGet（数组，arrayLength（数组）-1）】。\n",
            encoding="utf-8",
        )
        array_effect_function_artifact = temp / "runtime-array-effect-function"
        checked = run_unmodified("check", str(array_effect_function_source))
        built = run_unmodified(
            "build",
            str(array_effect_function_source),
            "-o",
            str(array_effect_function_artifact),
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("显式 array.write 效果的嵌套函数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                array_effect_function_artifact,
                required_imports=("_mmap", "_munmap"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        array_effect_function_run = subprocess.run(
            [str(array_effect_function_artifact)],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            array_effect_function_run.returncode != 0
            or array_effect_function_run.stdout != "42\n"
            or array_effect_function_run.stderr
        ):
            sys.stderr.write(array_effect_function_run.stdout + array_effect_function_run.stderr)
            print("显式 array.write 效果的嵌套函数结果不符", file=sys.stderr)
            return 1

        for label, fixture_name, supplied, expected in [
            (
                "运行时输入初始化变量",
                "p1b-runtime-read-variable.serica",
                "-314\r\n",
                "-314\n",
            ),
            (
                "运行时输入赋值变量",
                "p1b-runtime-read-assignment.serica",
                "9223372036854775807\n",
                "9223372036854775807\n",
            ),
            (
                "重复读取复用解析缓冲区",
                "p1b-runtime-read-multiple.serica",
                "41\n-27\r\n",
                "-27\n",
            ),
        ]:
            source = ROOT / "examples" / fixture_name
            artifact = temp / fixture_name.removesuffix(".serica")
            checked = run("check", str(source))
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label}的检查或构建失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_variable_storage=True,
                    require_runtime_input=True,
                    required_instructions=("ldrb\tw25, [x21]", "mul\tx23, x23, x26"),
                )
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label}结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，"
                    f"退出码={executed.returncode}",
                    file=sys.stderr,
                )
                return 1
            if fixture_name == "p1b-runtime-read-assignment.serica":
                rejected = subprocess.run(
                    [str(artifact)],
                    cwd=ROOT,
                    input="1x\n",
                    text=True,
                    capture_output=True,
                    check=False,
                )
                if rejected.returncode == 0 or rejected.stdout or rejected.stderr:
                    sys.stderr.write(rejected.stdout + rejected.stderr)
                    print("变量赋值期间的非法输入未以非零退出且保持无输出", file=sys.stderr)
                    return 1

        for index, (label, source_text, supplied, expected, instruction, has_variable) in enumerate(DYNAMIC_CASES):
            source = temp / f"dynamic-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"dynamic-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 的运行时语义检查或构建失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_variable_storage=has_variable,
                    require_runtime_input=True,
                    required_instructions=() if "array" in label else instruction if isinstance(instruction, tuple) else (instruction,),
                    required_imports=("_mmap", "_munmap") if "array" in label else (),
                )
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，"
                    f"退出码={executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, (label, source_text, supplied, expected_code) in enumerate(RESULT_FAILURE_CASES):
            source = temp / f"result-failure-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"result-failure-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成原生失败传播，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=not ("成功（『" in source_text and "输出=结果值（结果）" in source_text),
                    require_variable_storage=True,
                    require_runtime_input="readInteger（" in source_text or "readText（" in source_text,
                )
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1
            failed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if failed.returncode != expected_code or failed.stdout or failed.stderr:
                sys.stderr.write(failed.stdout + failed.stderr)
                print(
                    f"{label} 的失败传播不符：期望退出 {expected_code} 且无输出，"
                    f"实际 stdout={failed.stdout!r}，退出码={failed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, (label, source_text, supplied, expected) in enumerate(BLOCK_FUNCTION_CASES):
            source = temp / f"block-function-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"block-function-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成块体函数产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer="textConcat" not in source_text,
                    require_variable_storage=(
                        "变量 " in source_text and "textConcat" not in source_text
                    ),
                    require_runtime_input="readInteger（" in source_text or "readText（" in source_text,
                    required_imports=(
                        ("_mmap", "_munmap")
                        if "textConcat" in source_text or "结果值（" in source_text
                        else ()
                    ),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)], cwd=ROOT, input=supplied, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，"
                    f"退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, (label, source_text, supplied, expected) in enumerate(RUNTIME_BLOCK_LOOP_CASES):
            source = temp / f"runtime-block-loop-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"runtime-block-loop-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成运行时块循环函数产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=True,
                    require_variable_storage=True,
                    require_runtime_input=True,
                    required_instructions=("bl\t", "ret"),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)], cwd=ROOT, input=supplied, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，"
                    f"退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, case in enumerate(RUNTIME_FUNCTION_RESULT_RECOVERY_CASES):
            label, source_text, supplied, expected = case[:4]
            expected_code = case[4] if len(case) > 4 else 0
            source = temp / f"runtime-function-result-recovery-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            reject = next((item for item in case[4:] if isinstance(item, str) and item.startswith("S-")), None)
            if reject:
                checked = run_unmodified("check", str(source))
                if checked.returncode == 0 or reject not in (checked.stdout + checked.stderr):
                    sys.stderr.write(checked.stdout + checked.stderr)
                    print(f"{label} 应被静态拒绝为 {reject}", file=sys.stderr)
                    return 1
                continue
            checked = run("check", str(source))
            artifact = temp / f"runtime-function-result-recovery-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成块体函数结果恢复产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=True,
                    require_variable_storage=True,
                    require_runtime_input=True,
                    required_imports=("_mmap", "_munmap") if "变量 中途=" in source_text else (),
                    required_instructions=("bl\t", "ret"),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)], cwd=ROOT, input=supplied, text=True, capture_output=True, check=False
            )
            if executed.returncode != expected_code or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 stdout={expected!r}、退出码={expected_code}，"
                    f"实际 stdout={executed.stdout!r}、退出码={executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, case in enumerate(RUNTIME_TEXT_PARAMETER_CASES + OWNED_TEXT_RETURN_CASES + OWNED_TEXT_RETURN_STORAGE_CASES):
            label, source_text, supplied, expected = case[:4]
            expected_code = case[4] if len(case) > 4 else 0
            source = temp / f"runtime-text-parameter-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"runtime-text-parameter-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成运行时文本参数产物，但编译失败", file=sys.stderr)
                return 1
            try:
                required_instructions = ["blr\tx16"]
                if "readText（" in source_text:
                    required_instructions.append("ldr\tx21, [x19")
                owned_text_return = index >= len(RUNTIME_TEXT_PARAMETER_CASES)
                verify_native_console_artifact(
                    artifact,
                    require_integer=not owned_text_return,
                    require_variable_storage=False,
                    require_runtime_input=(
                        "readText（" in source_text or "readInteger（" in source_text
                    ),
                    required_imports=(
                        ("_mmap", "_munmap")
                        if owned_text_return
                        or source_declares_runtime_text_variable(source_text)
                        or label
                        in {
                            "copied text parameter length",
                            "copied text parameter comparison",
                            "copied text parameter grows through concatenation",
                            "independent copied text locals survive source reassignment",
                            "copied text parameter substring",
                            "copied supplementary scalar text parameter",
                            "early return cleans initialized copied text",
                        }
                        else ()
                    ),
                    required_instructions=tuple(required_instructions),
                )
                if index >= len(RUNTIME_TEXT_PARAMETER_CASES) + len(OWNED_TEXT_RETURN_CASES):
                    disassembly = subprocess.run(
                        ["otool", "-tV", str(artifact)], text=True, capture_output=True, check=False
                    )
                    if disassembly.returncode != 0:
                        raise RuntimeError(f"无法反汇编文本返回存储产物\n{disassembly.stdout}{disassembly.stderr}")
                    instructions = [
                        line.split("\t", 1)[1].strip()
                        for line in disassembly.stdout.splitlines()
                        if "\t" in line
                    ]
                    save_positions = [
                        position
                        for position, instruction in enumerate(instructions)
                        if re.fullmatch(r"str\s+x1, \[sp(?:, #[^\]]+)?\]", instruction)
                    ]
                    call_positions = [
                        position
                        for position, instruction in enumerate(instructions)
                        if re.match(r"bl\s", instruction)
                    ]
                    if not save_positions or not call_positions or not any(
                        0 < save - call <= 3
                        for call in call_positions
                        for save in save_positions
                    ):
                        raise RuntimeError("文本返回存储产物未在函数调用后立即保存 x1 返回指针")
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)], cwd=ROOT, input=supplied, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，"
                    f"退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        text_parameter_failures = [
            (
                "integer passed to text parameter",
                "函数 文本长度（来源）【返回 textLength（来源）。】。\n"
                "程序=【能力=array（『stdout』），输出=文本长度（1）】。\n",
                "S-3062",
            ),
            (
                "text passed to runtime integer parameter",
                "函数 限制（数值）【凡（数值>0）【返回 数值。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』），输出=限制（『文本』）】。\n",
                "S-3062",
            ),
            (
                "text parameter assignment",
                "函数 改写（来源）【赋值 来源=『新』。返回 textLength（来源）。】。\n"
                "程序=【能力=array（『stdout』），输出=改写（『旧』）】。\n",
                "S-3068",
            ),
            (
                "text parameter arithmetic escape",
                "函数 错误（来源）【返回 textLength（来源）+来源。】。\n"
                "程序=【能力=array（『stdout』），输出=错误（『文本』）】。\n",
                "S-3023",
            ),
            (
                "five text parameters exceed register budget",
                "函数 合计（甲，乙，丙，丁，戊）【"
                "返回 textLength（甲）+textLength（乙）+textLength（丙）+textLength（丁）+textLength（戊）。】。\n"
                "程序=【能力=array（『stdout』），输出=合计（『a』，『b』，『c』，『d』，『e』）】。\n",
                "S-3062",
            ),
            (
                "hidden input inside text parameter function",
                "函数 长度（来源）【返回 textLength（readText（））。】。\n"
                "程序=【能力=array（『stdout』），输出=长度（『a』）】。\n",
                "S-4106",
            ),
            (
                "text parameter ordering against integer",
                "函数 错误（来源）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
                "若（equals（来源，『行简』））【赋值 守卫=守卫+0。】。"
                "若（来源<1）【返回 1。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』），输出=错误（『行简』）】。\n",
                "S-3023",
            ),
            (
                "text parameter equality against integer",
                "函数 错误（来源）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
                "若（equals（来源，『行简』））【赋值 守卫=守卫+0。】。"
                "若（equals（来源，1））【返回 1。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』），输出=错误（『行简』）】。\n",
                "S-3015",
            ),
            (
                "text parameter return escape",
                "函数 错误（来源）【变量 守卫=0。凡（守卫<1）【赋值 守卫=守卫+1。】。"
                "若（equals（来源，『行简』））【返回 来源。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』），输出=错误（『行简』）】。\n",
                "S-3068",
            ),
            (
                "text local returned before initialization",
                "函数 逃逸（来源）【变量 长度=textLength（来源）。返回 副本。变量 副本=来源。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=逃逸（输入）】。\n",
                "S-3068",
            ),
            (
                "text local inside loop",
                "函数 循环内（来源）【变量 长度=textLength（来源）。变量 索引=0。"
                "凡（索引<1）【变量 副本=来源。赋值 索引=索引+1。】。返回 长度+索引。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=循环内（输入）】。\n",
                "S-3068",
            ),
            (
                "text local inside branch",
                "函数 分支内（来源）【变量 长度=textLength（来源）。"
                "若（长度>=0）【变量 副本=来源。返回 textLength（副本）。】。返回 0。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=分支内（输入）】。\n",
                "S-3068",
            ),
            (
                "hidden input initializes text local",
                "函数 隐藏（来源）【变量 长度=textLength（来源）。变量 副本=readText（）。"
                "返回 长度+textLength（副本）。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=隐藏（输入）】。\n",
                "S-4106",
            ),
            (
                "text parameter escapes into array",
                "函数 容器（来源）【变量 长度=textLength（来源）。变量 表=array（『头』）。"
                "赋值 表=arrayPush（表，来源）。返回 长度+arrayLength（表）。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』，『array.write』），输出=容器（输入）】。\n",
                "S-3068",
            ),
            (
                "owned text return in a condition",
                "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
                "变量 输入=readText（）。\n若（欢迎（输入））【】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=0】。\n",
                "S-3023",
            ),
            (
                "owned text return borrowed length in a condition",
                "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
                "变量 输入=readText（）。\n若（textLength（欢迎（输入））>0）【】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=0】。\n",
                "S-3023",
            ),
            (
                "owned text return borrowed equality in a statement",
                "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
                "变量 输入=readText（）。\n"
                "若（equals（欢迎（输入），『行简』））【】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=0】。\n",
                "S-3023",
            ),
            (
                "owned text return in another function",
                "函数 欢迎（来源）【变量 副本=来源。变量 长度=textLength（副本）。返回 副本。】。\n"
                "函数 代理（来源）=欢迎（来源）。\n"
                "程序=【能力=array（『stdout』），输出=1】。\n",
                "S-3023",
            ),
            (
                "owned text return inside a function body",
                "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
                "函数 代理（来源）【变量 结果=欢迎（来源）。返回 textLength（结果）。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=代理（输入）】。\n",
                "S-3023",
            ),
            (
                "owned text return nested in root expression",
                "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=textConcat（欢迎（输入），『尾』）】。\n",
                "S-3044",
            ),
            (
                "text parameter returned without owned copy",
                "函数 直返（来源）【返回 来源。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=直返（输入）】。\n",
                "S-3068",
            ),
            (
                "owned text return from a branch",
                "函数 分支（来源）【变量 副本=来源。"
                "若（equals（副本，『行简』））【返回 副本。】。返回 副本。】。\n"
                "变量 输入=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=分支（输入）】。\n",
                "S-3068",
            ),
        ]
        for name, source_text, expected_code in text_parameter_failures:
            invalid = temp / f"{name}.serica"
            invalid.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = (
                    run(command, str(invalid), "-o", str(temp / f"{name}-{command}"))
                    if command == "build"
                    else run(command, str(invalid))
                )
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(f"错误输入 {name} 的 {command} 未返回 {expected_code}", file=sys.stderr)
                    return 1

        invalid_text_parameter_input = temp / "runtime-text-parameter-invalid-input.serica"
        invalid_text_parameter_input.write_text(
            next(
                source_text
                for label, source_text, _supplied, _expected in RUNTIME_TEXT_PARAMETER_CASES
                if label == "copied text parameter length"
            ),
            encoding="utf-8",
        )
        invalid_text_parameter_artifact = temp / "runtime-text-parameter-invalid-input"
        checked = run("check", str(invalid_text_parameter_input))
        built = run(
            "build",
            str(invalid_text_parameter_input),
            "-o",
            str(invalid_text_parameter_artifact),
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时文本参数无效 UTF-8 反例构建失败", file=sys.stderr)
            return 1
        rejected = subprocess.run(
            [str(invalid_text_parameter_artifact)],
            cwd=ROOT,
            input=b"\xed\xa0\x80\n",
            capture_output=True,
            check=False,
        )
        if rejected.returncode == 0 or rejected.stdout or rejected.stderr:
            sys.stderr.write(rejected.stdout.decode("utf-8", "replace") + rejected.stderr.decode("utf-8", "replace"))
            print("运行时文本参数的无效 UTF-8 未以非零退出且保持无输出", file=sys.stderr)
            return 1

        invalid_text_return_input = temp / "runtime-text-return-invalid-input.serica"
        invalid_text_return_input.write_text(
            next(
                case[1]
                for case in OWNED_TEXT_RETURN_STORAGE_CASES
                if case[0] == "owned text return initializes a variable"
            ),
            encoding="utf-8",
        )
        invalid_text_return_artifact = temp / "runtime-text-return-invalid-input"
        checked = run("check", str(invalid_text_return_input))
        built = run(
            "build",
            str(invalid_text_return_input),
            "-o",
            str(invalid_text_return_artifact),
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时文本所有权返回的无效 UTF-8 反例构建失败", file=sys.stderr)
            return 1
        rejected = subprocess.run(
            [str(invalid_text_return_artifact)],
            cwd=ROOT,
            input=b"\xed\xa0\x80\n",
            capture_output=True,
            check=False,
        )
        if rejected.returncode == 0 or rejected.stdout or rejected.stderr:
            sys.stderr.buffer.write(rejected.stdout + rejected.stderr)
            print("运行时文本所有权返回未在无效 UTF-8 时保持零输出并非零退出", file=sys.stderr)
            return 1

        for index, (label, source_text, supplied, expected) in enumerate(RUNTIME_FUNCTION_ABI_CASES):
            source = temp / f"runtime-function-abi-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"runtime-function-abi-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成递归运行时函数产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=True,
                    require_variable_storage=True,
                    require_runtime_input=True,
                    required_instructions=("bl\t", "ret"),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，"
                    f"退出码 {executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, case in enumerate(RUNTIME_RESULT_ABI_CASES):
            label, source_text, supplied, expected = case[:4]
            expected_code = case[4] if len(case) > 4 else 0
            source = temp / f"runtime-result-abi-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            if len(case) > 5 and case[5] == "check":
                if checked.returncode == 0 or expected not in checked.stdout:
                    sys.stderr.write(checked.stdout + checked.stderr)
                    print(f"{label} 应被静态拒绝为 {expected}", file=sys.stderr)
                    return 1
                continue
            artifact = temp / f"runtime-result-abi-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成递归运行时结果函数产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=expected_code == 0,
                    require_variable_storage=True,
                    require_runtime_input="readInteger" in source_text,
                    required_imports=("_mmap", "_munmap") if "变量 副本=" in source_text else (),
                    required_instructions=("bl\t", "ret"),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != expected_code or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望退出 {expected_code} 和 stdout={expected!r}，"
                    f"实际退出 {executed.returncode} 和 stdout={executed.stdout!r}",
                    file=sys.stderr,
                )
                return 1

        for index, case in enumerate(RUNTIME_TEXT_RESULT_ABI_CASES):
            label, source_text, supplied, expected = case[:4]
            expected_code = case[4] if len(case) > 4 else 0
            source = temp / f"runtime-text-result-abi-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"runtime-text-result-abi-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成递归文本结果函数产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=False,
                    require_variable_storage=True,
                    require_runtime_input=True,
                    required_instructions=("bl\t", "ret"),
                    required_imports=("_mmap", "_munmap") if "变量 目标=结果值（源）" in source_text else (),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if executed.returncode != expected_code or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望退出 {expected_code} 和 stdout={expected!r}，"
                    f"实际退出 {executed.returncode} 和 stdout={executed.stdout!r}",
                    file=sys.stderr,
                )
                return 1

        for index, case in enumerate(RESULT_FUNCTION_CASES):
            label, source_text, supplied, expected = case[:4]
            expected_code = case[4] if len(case) > 4 else 0
            source = temp / f"result-function-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"result-function-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成函数结果边界产物，但编译失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_integer=expected_code == 0 and expected.rstrip("\n").lstrip("-").isdigit(),
                    require_variable_storage=True,
                    require_runtime_input="readInteger（" in source_text or "readText（" in source_text,
                    required_imports=(
                        ("_mmap", "_munmap")
                        if "变量 显示=0" in source_text
                        or "成功（readText" in source_text
                        or "成功（textConcat" in source_text
                        or "变量 结果=ifElse" in source_text
                        or "选择（）" in source_text
                        or ("变量 副本=" in source_text and "变量 原结果=失败（" not in source_text)
                        else ()
                    ),
                )
            except RuntimeError as failure:
                print(f"{label}：{failure}", file=sys.stderr)
                return 1
            if label in {
                "effectful function initializes a text result local",
                "effectful concatenated text result local",
                "effectful sliced text result local",
            }:
                first_image = artifact.read_bytes()
                for _ in range(7):
                    rebuilt = run("build", str(source), "-o", str(artifact))
                    if rebuilt.returncode != 0:
                        sys.stderr.write(rebuilt.stdout + rebuilt.stderr)
                        print(f"{label} 同路径重建失败", file=sys.stderr)
                        return 1
                    if artifact.read_bytes() != first_image:
                        print(f"{label} 同路径重建字节不同", file=sys.stderr)
                        return 1
            executed = subprocess.run(
                [str(artifact)], cwd=ROOT, input=supplied, text=True, capture_output=True, check=False
            )
            if executed.returncode != expected_code or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 stdout={expected!r}、退出码={expected_code}，"
                    f"实际 stdout={executed.stdout!r}、退出码={executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        for index, (label, source_text, supplied, expected) in enumerate(MAP_CASES):
            source = temp / f"map-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run_unmodified("check", str(source))
            artifact = temp / f"map-{index}"
            built = run_unmodified("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 的映射检查或构建失败", file=sys.stderr)
                return 1
            try:
                verify_native_console_artifact(
                    artifact,
                    require_variable_storage=True,
                    require_runtime_input="readInteger（" in source_text,
                    required_imports=("_mmap", "_munmap"),
                )
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1
            executed = subprocess.run(
                [str(artifact)], cwd=ROOT, input=supplied, text=True, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.write(executed.stdout + executed.stderr)
                print(
                    f"{label} 结果不符：期望 {expected!r}，实际 stdout={executed.stdout!r}，退出码={executed.returncode}",
                    file=sys.stderr,
                )
                return 1

        text_artifact = temp / "runtime-text"
        checked = run("check", str(ROOT / "examples" / "p1b-runtime-text.serica"))
        built = run("build", str(ROOT / "examples" / "p1b-runtime-text.serica"), "-o", str(text_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 UTF-8 文本子门无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                text_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw23, [x24]", "strb\tw23, [x24]", "b.hi"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1

        for label, supplied, expected in RUNTIME_TEXT_CASES:
            executed = subprocess.run(
                [str(text_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        for label, supplied in RUNTIME_TEXT_FAILURES:
            failed = subprocess.run(
                [str(text_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"运行时 UTF-8 文本 {label} 未以无输出非零退出", file=sys.stderr)
                return 1

        text_variable_artifact = temp / "runtime-text-variable"
        text_variable_source = temp / "runtime-text-variable.serica"
        text_variable_source.write_text(RUNTIME_TEXT_VARIABLE_FIXTURE, encoding="utf-8")
        checked = run("check", str(text_variable_source))
        built = run("build", str(text_variable_source), "-o", str(text_variable_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 UTF-8 文本变量无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                text_variable_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw23, [x24]", "strb\tw23, [x24]", "ldr\tx2, [x19, #0x8]", "ldr\tx22, [x19, #0x10]", "b.hs"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1

        for label, supplied, expected in RUNTIME_TEXT_VARIABLE_CASES:
            executed = subprocess.run(
                [str(text_variable_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本变量 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        multiple_text_source = temp / "runtime-text-multiple-variables.serica"
        multiple_text_source.write_text(
            "变量 首行=readText（）。\n变量 次行=readText（）。\n程序=【输出=首行】。\n",
            encoding="utf-8",
        )
        multiple_text_artifact = temp / "runtime-text-multiple-variables"
        checked = run("check", str(multiple_text_source))
        built = run("build", str(multiple_text_source), "-o", str(multiple_text_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("多个运行时文本变量无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                multiple_text_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("str\tx0, [x19]", "str\tx0, [x19, #0x18]"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        multiple_text_run = subprocess.run(
            [str(multiple_text_artifact)],
            cwd=ROOT,
            input="保留行\n次行\n".encode("utf-8"),
            capture_output=True,
            check=False,
        )
        if multiple_text_run.returncode != 0 or multiple_text_run.stdout != "保留行\n".encode("utf-8") or multiple_text_run.stderr:
            sys.stderr.buffer.write(multiple_text_run.stdout + multiple_text_run.stderr)
            print("多个文本变量未保留各自独立的输入缓冲区", file=sys.stderr)
            return 1

        for label, supplied in RUNTIME_TEXT_VARIABLE_FAILURES:
            failed = subprocess.run(
                [str(text_variable_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"运行时 UTF-8 文本变量 {label} 未以无输出非零退出", file=sys.stderr)
                return 1

        concat_source = ROOT / "examples" / "p1b-runtime-text-concat.serica"
        concat_artifact = temp / "runtime-text-concat"
        checked = run("check", str(concat_source))
        built = run("build", str(concat_source), "-o", str(concat_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 UTF-8 文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                concat_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw0, [x1]", "strb\tw0, [x1]", "b.gt"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1

        for label, supplied, expected in RUNTIME_TEXT_CONCAT_CASES:
            executed = subprocess.run(
                [str(concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本拼接 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        alias_source = temp / "runtime-text-concat-alias.serica"
        alias_source.write_text(
            "变量 前项=readText（）。\n变量 后项=readText（）。\n"
            "变量 组合=textConcat（前项，后项）。\n"
            "赋值 前项=textConcat（前项，后项）。\n"
            "赋值 后项=textConcat（前项，后项）。\n程序=【输出=后项】。\n",
            encoding="utf-8",
        )
        alias_artifact = temp / "runtime-text-concat-alias"
        checked = run("check", str(alias_source))
        built = run("build", str(alias_source), "-o", str(alias_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("写回拼接结果到源文本变量失败", file=sys.stderr)
            return 1
        alias_run = subprocess.run(
            [str(alias_artifact)], cwd=ROOT, input=b"same\n-buffer\n", capture_output=True, check=False, timeout=5
        )
        if alias_run.returncode != 0 or alias_run.stdout != b"same-buffer-buffer\n" or alias_run.stderr:
            sys.stderr.buffer.write(alias_run.stdout + alias_run.stderr)
            print("拼接结果写回输入变量时发生别名破坏", file=sys.stderr)
            return 1

        for label, supplied in RUNTIME_TEXT_CONCAT_FAILURES:
            failed = subprocess.run(
                [str(concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=120
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"运行时 UTF-8 文本拼接 {label} 未以无输出非零退出", file=sys.stderr)
                return 1
        maximum_concat_input = b"a" * (MAX_RUNTIME_TEXT_CAPACITY - 2) + b"\nx\n"
        maximum_concat_expected = b"a" * (MAX_RUNTIME_TEXT_CAPACITY - 2) + b"x\n"
        maximum_concat = subprocess.run(
            [str(concat_artifact)], cwd=ROOT, input=maximum_concat_input, capture_output=True, check=False, timeout=120
        )
        if maximum_concat.returncode != 0 or maximum_concat.stdout != maximum_concat_expected or maximum_concat.stderr:
            sys.stderr.buffer.write(maximum_concat.stdout[:256] + maximum_concat.stderr)
            print("变量文本拼接的 16 MiB−1 字节最大结果未通过", file=sys.stderr)
            return 1

        concat_output_source = ROOT / "examples" / "p1b-runtime-text-concat-output.serica"
        concat_output_artifact = temp / "runtime-text-concat-output"
        checked = run("check", str(concat_output_source))
        built = run("build", str(concat_output_source), "-o", str(concat_output_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("直接输出运行时文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                concat_output_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw28, [x27]", "cmp\tx23, #0x3f", "b.gt"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_OUTPUT_CASES:
            executed = subprocess.run(
                [str(concat_output_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"直接输出运行时文本拼接 {label} 结果不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_CONCAT_OUTPUT_FAILURES:
            failed = subprocess.run(
                [str(concat_output_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"直接输出运行时文本拼接 {label} 未以无输出非零退出", file=sys.stderr)
                return 1

        literal_concat_source = ROOT / "examples" / "p1b-runtime-text-concat-literals.serica"
        literal_concat_artifact = temp / "runtime-text-concat-literals"
        checked = run("check", str(literal_concat_source))
        built = run("build", str(literal_concat_source), "-o", str(literal_concat_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("静态/运行时文本拼接操作数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                literal_concat_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw28, [x27]", "strb\tw28, [x27]", "b.gt"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_LITERAL_CASES:
            executed = subprocess.run(
                [str(literal_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"静态/运行时文本拼接 {label} 结果不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_CONCAT_LITERAL_FAILURES:
            failed = subprocess.run(
                [str(literal_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"静态/运行时文本拼接 {label} 未在无输出时以非零退出", file=sys.stderr)
                return 1

        read_concat_source = ROOT / "examples" / "p1b-runtime-text-concat-read.serica"
        read_concat_artifact = temp / "runtime-text-concat-read"
        checked = run("check", str(read_concat_source))
        built = run("build", str(read_concat_source), "-o", str(read_concat_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("readText（）作为 textConcat 操作数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                read_concat_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw0, [x1]", "strb\tw0, [x1]", "b.gt"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_READ_CASES:
            executed = subprocess.run(
                [str(read_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"readText（）textConcat {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_CONCAT_READ_FAILURES:
            failed = subprocess.run(
                [str(read_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"readText（）textConcat {label} 未在输出前拒绝", file=sys.stderr)
                return 1

        read_slice_concat_source = ROOT / "examples" / "p1b-runtime-text-concat-read-slice.serica"
        read_slice_concat_artifact = temp / "runtime-text-concat-read-slice"
        checked = run("check", str(read_slice_concat_source))
        built = run("build", str(read_slice_concat_source), "-o", str(read_slice_concat_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("直接 readText 切片作为 textConcat 操作数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                read_slice_concat_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("b.gt",),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_READ_SLICE_CASES:
            executed = subprocess.run(
                [str(read_slice_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"直接 readText 切片拼接 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_CONCAT_READ_SLICE_FAILURES:
            failed = subprocess.run(
                [str(read_slice_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"直接 readText 切片拼接 {label} 未在输出前拒绝", file=sys.stderr)
                return 1

        comparison_programs = [
            (
                "runtime-text-equals.serica",
                "变量 左文=readText（）。\n变量 右文=readText（）。\n"
                "程序=【输出=ifElse（equals（左文，右文），1，0）】。\n",
                RUNTIME_TEXT_COMPARISON_CASES,
            ),
            (
                "runtime-text-not-equals.serica",
                "变量 左文=readText（）。\n变量 右文=readText（）。\n变量 命中=0。\n"
                "若（notEquals（左文，右文））【赋值 命中=1。】。\n程序=【输出=命中】。\n",
                RUNTIME_TEXT_NOT_EQUAL_CASES,
            ),
            (
                "runtime-text-equals-short-circuit.serica",
                "变量 左文=readText（）。\n变量 右文=readText（）。\n变量 命中=0。\n"
                "若（equals（左文，右文）且 readInteger（）==1）【赋值 命中=1。】。\n"
                "程序=【输出=命中】。\n",
                RUNTIME_TEXT_SHORT_CIRCUIT_CASES,
            ),
            (
                "p1b-runtime-text-equals-static.serica",
                None,
                RUNTIME_TEXT_STATIC_COMPARISON_CASES,
            ),
            (
                "runtime-text-equals-concat-and-slice.serica",
                "变量 文本=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（"
                "equals（readText（），『标记』）且 "
                "equals（textConcat（textConcat（textConcat（textConcat（文本，『甲』），『乙』），『丙』），『丁』），『行简甲乙丙丁』）且 "
                "equals（textConcat（文本，『甲』），readText（））且 "
                "notEquals（substring（文本，0，2），textConcat（文本，『错』）），42，0）】。\n",
                [("nested concat and slice", "行简\n标记\n行简甲\n".encode(), b"42\n")],
            ),
            (
                "runtime-text-equals-array-get.serica",
                "变量 文本数组=array（『占位』）。\n"
                "赋值 文本数组=arrayPush（文本数组，readText（））。\n"
                "程序=【能力=array（『stdout』，『stdin』，『array.write』），"
                "输出=ifElse（equals（arrayGet（文本数组，1），『标记』），42，0）】。\n",
                [("dynamic text array element matches", "标记\n".encode(), b"42\n"),
                 ("dynamic text array element differs", "其他\n".encode(), b"0\n")],
            ),
        ]
        for fixture_name, source_text, cases in comparison_programs:
            source = ROOT / "examples" / fixture_name if source_text is None else temp / fixture_name
            if source_text is not None:
                source.write_text(source_text, encoding="utf-8")
            artifact = temp / fixture_name.removesuffix(".serica")
            checked = run("check", str(source))
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{fixture_name} 的检查或构建失败", file=sys.stderr)
                return 1
            try:
                verify_native_text_storage_artifact(
                    artifact,
                    require_integer=False,
                    require_runtime_input=True,
                    required_instructions=("ldrb\tw27, [x26]", "cmp\tx27, x28", "b.ne"),
                )
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1
            for label, supplied, expected in cases:
                executed = subprocess.run(
                    [str(artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False
                )
                if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                    sys.stderr.buffer.write(executed.stdout + executed.stderr)
                    print(f"{fixture_name} {label} 的文本比较结果不符：{executed.stdout!r}", file=sys.stderr)
                    return 1

        ordering_source = ROOT / "examples" / "p1b-runtime-text-order.serica"
        ordering_artifact = temp / "runtime-text-ordering"
        checked = run("check", str(ordering_source))
        built = run("build", str(ordering_source), "-o", str(ordering_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 UTF-8 文本排序比较无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                ordering_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw27, [x26]", "cmp\tx27, x28", "b.hs", "b.hi", "b.ls", "b.lo"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_ORDERING_CASES:
            executed = subprocess.run(
                [str(ordering_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本排序比较 {label} 结果不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        order_expression_source = ROOT / "examples" / "p1b-runtime-text-order-expressions.serica"
        order_expression_artifact = temp / "runtime-text-order-expressions"
        checked = run("check", str(order_expression_source))
        built = run("build", str(order_expression_source), "-o", str(order_expression_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时文本表达式排序比较无法检查或构建", file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_ORDER_EXPRESSION_CASES:
            executed = subprocess.run(
                [str(order_expression_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时文本表达式排序比较 {label} 结果不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        text_length_source = ROOT / "examples" / "p1b-runtime-text-length.serica"
        text_length_artifact = temp / "runtime-text-length"
        checked = run("check", str(text_length_source))
        built = run("build", str(text_length_source), "-o", str(text_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 UTF-8 文本码位长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                text_length_artifact,
                require_runtime_input=True,
                required_instructions=("ldrb\tw26, [x25]", "cmp\tx26, #0x80", "cmp\tx26, #0xc0", "b.hs", "b.lo"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_LENGTH_CASES:
            executed = subprocess.run(
                [str(text_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本码位长度 {label} 结果不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_FAILURES:
            executed = subprocess.run(
                [str(text_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode == 0 or executed.stdout or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本码位长度未在输出前拒绝 {label}", file=sys.stderr)
                return 1

        long_text_length_source = temp / "runtime-text-long-length.serica"
        long_text_length_source.write_text(
            "变量 文本=readText（）。\n程序=【输出=textLength（文本）】。\n", encoding="utf-8"
        )
        long_text_length_artifact = temp / "runtime-text-long-length"
        checked = run("check", str(long_text_length_source))
        built = run("build", str(long_text_length_source), "-o", str(long_text_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态增长文本的 UTF-8 码位长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                long_text_length_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("cmp\tx26, #0x80", "cmp\tx26, #0xc0"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        long_text = "𠮷" * 100
        long_text_run = subprocess.run(
            [str(long_text_length_artifact)],
            cwd=ROOT,
            input=(long_text + "\r\n").encode("utf-8"),
            capture_output=True,
            check=False,
            timeout=15,
        )
        if long_text_run.returncode != 0 or long_text_run.stdout != b"100\n" or long_text_run.stderr:
            sys.stderr.buffer.write(long_text_run.stdout + long_text_run.stderr)
            print("多次扩容后的文本码位长度不符", file=sys.stderr)
            return 1
        malformed_long_text = subprocess.run(
            [str(long_text_length_artifact)],
            cwd=ROOT,
            input=b"x" * 100 + b"\xed\xa0\x80\n",
            capture_output=True,
            check=False,
            timeout=15,
        )
        if malformed_long_text.returncode == 0 or malformed_long_text.stdout or malformed_long_text.stderr:
            sys.stderr.buffer.write(malformed_long_text.stdout + malformed_long_text.stderr)
            print("动态缓冲区中的非法 UTF-8 未在输出前拒绝", file=sys.stderr)
            return 1
        maximum_text = subprocess.run(
            [str(long_text_length_artifact)],
            cwd=ROOT,
            input=b"a" * (MAX_RUNTIME_TEXT_CAPACITY - 1) + b"\n",
            capture_output=True,
            check=False,
            timeout=120,
        )
        if (
            maximum_text.returncode != 0
            or maximum_text.stdout != f"{MAX_RUNTIME_TEXT_CAPACITY - 1}\n".encode("ascii")
            or maximum_text.stderr
        ):
            sys.stderr.buffer.write(maximum_text.stdout + maximum_text.stderr)
            print("动态文本最大有效输入长度未通过", file=sys.stderr)
            return 1
        oversized_text = subprocess.run(
            [str(long_text_length_artifact)],
            cwd=ROOT,
            input=b"a" * MAX_RUNTIME_TEXT_CAPACITY + b"\n",
            capture_output=True,
            check=False,
            timeout=120,
        )
        if oversized_text.returncode == 0 or oversized_text.stdout or oversized_text.stderr:
            sys.stderr.buffer.write(oversized_text.stdout + oversized_text.stderr)
            print("动态文本超过 16 MiB 容量时未在输出前拒绝", file=sys.stderr)
            return 1

        read_text_length_source = ROOT / "examples" / "p1b-runtime-read-text-length.serica"
        read_text_length_artifact = temp / "runtime-read-text-length"
        checked = run("check", str(read_text_length_source))
        built = run("build", str(read_text_length_source), "-o", str(read_text_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("直接 readText（）的 UTF-8 文本码位长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                read_text_length_artifact,
                require_runtime_input=True,
                required_instructions=("ldrb\tw26, [x25]", "cmp\tx26, #0x80", "cmp\tx26, #0xc0", "b.hs", "b.lo"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_LENGTH_CASES:
            executed = subprocess.run(
                [str(read_text_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"直接 readText（）的 UTF-8 文本码位长度 {label} 结果不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_FAILURES:
            executed = subprocess.run(
                [str(read_text_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode == 0 or executed.stdout or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"直接 readText（）的文本码位长度未在输出前拒绝 {label}", file=sys.stderr)
                return 1

        slice_artifact = temp / "runtime-text-slice"
        slice_source = ROOT / "examples" / "p1b-runtime-text-slice.serica"
        checked = run("check", str(slice_source))
        built = run("build", str(slice_source), "-o", str(slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 UTF-8 文本切片无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                slice_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw28, [x27]", "b.hs"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_SLICE_CASES:
            try:
                executed = subprocess.run(
                    [str(slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
                )
            except subprocess.TimeoutExpired:
                print(f"运行时 UTF-8 文本切片 {label} 未终止", file=sys.stderr)
                return 1
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时 UTF-8 文本切片 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        read_slice_artifact = temp / "runtime-text-read-slice"
        read_slice_source = ROOT / "examples" / "p1b-runtime-text-read-slice.serica"
        checked = run("check", str(read_slice_source))
        built = run("build", str(read_slice_source), "-o", str(read_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("直接 readText（）的 UTF-8 文本切片无法检查或构建", file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_READ_SLICE_CASES:
            executed = subprocess.run(
                [str(read_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"直接 readText（）的 UTF-8 文本切片 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_READ_SLICE_FAILURES:
            failed = subprocess.run(
                [str(read_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"直接 readText（）的 UTF-8 文本切片未在输出前拒绝 {label}", file=sys.stderr)
                return 1

        for label, supplied in RUNTIME_TEXT_SLICE_FAILURES:
            failed = subprocess.run(
                [str(slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"运行时 UTF-8 文本切片 {label} 未以无输出非零退出", file=sys.stderr)
                return 1

        long_slice_source = temp / "runtime-text-slice-long.serica"
        long_slice_source.write_text(
            "变量 文本=readText（）。\n程序=【输出=substring（文本，30，32）】。\n", encoding="utf-8"
        )
        long_slice_artifact = temp / "runtime-text-slice-long"
        checked = run("check", str(long_slice_source))
        built = run("build", str(long_slice_source), "-o", str(long_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态文本后段切片无法检查或构建", file=sys.stderr)
            return 1
        long_slice_run = subprocess.run(
            [str(long_slice_artifact)],
            cwd=ROOT,
            input=("中" * 30 + "𠮷Z\n").encode("utf-8"),
            capture_output=True,
            check=False,
            timeout=15,
        )
        if long_slice_run.returncode != 0 or long_slice_run.stdout != "𠮷Z\n".encode("utf-8") or long_slice_run.stderr:
            sys.stderr.buffer.write(long_slice_run.stdout + long_slice_run.stderr)
            print("动态文本超过 63 字节偏移后的码位切片不符", file=sys.stderr)
            return 1

        alias_slice_source = temp / "runtime-text-slice-alias.serica"
        alias_slice_source.write_text(
            "变量 文本=readText（）。\n赋值 文本=substring（文本，1，3）。\n程序=【输出=文本】。\n",
            encoding="utf-8",
        )
        alias_slice_artifact = temp / "runtime-text-slice-alias"
        checked = run("check", str(alias_slice_source))
        built = run("build", str(alias_slice_source), "-o", str(alias_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时文本切片写回原变量失败", file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_ALIAS_SLICE_CASES:
            executed = subprocess.run(
                [str(alias_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时文本切片 {label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1

        empty_slice_source = temp / "runtime-text-slice-empty.serica"
        empty_slice_source.write_text(
            "变量 文本=readText（）。\n程序=【输出=substring（文本，1，1）】。\n",
            encoding="utf-8",
        )
        empty_slice_artifact = temp / "runtime-text-slice-empty"
        checked = run("check", str(empty_slice_source))
        built = run("build", str(empty_slice_source), "-o", str(empty_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("空运行时文本切片无法检查或构建", file=sys.stderr)
            return 1
        empty_slice_run = subprocess.run(
            [str(empty_slice_artifact)], cwd=ROOT, input=b"x\n", capture_output=True, check=False, timeout=5
        )
        if empty_slice_run.returncode != 0 or empty_slice_run.stdout != b"\n" or empty_slice_run.stderr:
            sys.stderr.buffer.write(empty_slice_run.stdout + empty_slice_run.stderr)
            print("空运行时文本切片输出不符", file=sys.stderr)
            return 1

        max_slice_source = ROOT / "examples" / "p1b-runtime-text-slice-max.serica"
        max_slice_artifact = temp / "runtime-text-slice-max-index"
        checked = run("check", str(max_slice_source))
        built = run("build", str(max_slice_source), "-o", str(max_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("最大码位索引切片无法检查或构建", file=sys.stderr)
            return 1
        max_slice_run = subprocess.run(
            [str(max_slice_artifact)], cwd=ROOT, input=b"x" * 63 + b"\n", capture_output=True, check=False, timeout=5
        )
        if max_slice_run.returncode != 0 or max_slice_run.stdout != b"x\n" or max_slice_run.stderr:
            sys.stderr.buffer.write(max_slice_run.stdout + max_slice_run.stderr)
            print("最大码位索引 62..63 切片输出不符", file=sys.stderr)
            return 1

        dynamic_slice_source = ROOT / "examples" / "p1b-runtime-text-slice-dynamic.serica"
        dynamic_slice_artifact = temp / "runtime-text-slice-dynamic-index"
        checked = run("check", str(dynamic_slice_source))
        built = run("build", str(dynamic_slice_source), "-o", str(dynamic_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时整数切片索引无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                dynamic_slice_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("mov\tx6, x0", "mov\tx7, x0", "cmp\tx7, x6", "b.lt"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_DYNAMIC_SLICE_CASES:
            executed = subprocess.run(
                [str(dynamic_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"{label} 切片输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_DYNAMIC_SLICE_FAILURES:
            failed = subprocess.run(
                [str(dynamic_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"{label} 未以无输出非零退出", file=sys.stderr)
                return 1

        long_slice_assignment_source = temp / "runtime-text-slice-assignment-long.serica"
        long_slice_assignment_source.write_text(
            "变量 文本=readText（）。\n赋值 文本=substring（文本，0，63）。\n程序=【输出=文本】。\n",
            encoding="utf-8",
        )
        long_slice_assignment_artifact = temp / "runtime-text-slice-assignment-long"
        checked = run("check", str(long_slice_assignment_source))
        built = run("build", str(long_slice_assignment_source), "-o", str(long_slice_assignment_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("长 UTF-8 切片写回动态文本变量无法检查或构建", file=sys.stderr)
            return 1
        long_slice_input = ("𠮷" * 100 + "\n").encode("utf-8")
        long_slice_expected = ("𠮷" * 63 + "\n").encode("utf-8")
        long_slice_assignment = subprocess.run(
            [str(long_slice_assignment_artifact)], cwd=ROOT, input=long_slice_input, capture_output=True, check=False, timeout=5
        )
        if long_slice_assignment.returncode != 0 or long_slice_assignment.stdout != long_slice_expected or long_slice_assignment.stderr:
            sys.stderr.buffer.write(long_slice_assignment.stdout[:256] + long_slice_assignment.stderr)
            print("长 UTF-8 切片写回动态文本变量的结果不符", file=sys.stderr)
            return 1

        concat_slice_operand_source = ROOT / "examples" / "p1b-runtime-text-concat-slice-operand.serica"
        concat_slice_operand_artifact = temp / "runtime-text-concat-slice-operand"
        checked = run("check", str(concat_slice_operand_source))
        built = run("build", str(concat_slice_operand_source), "-o", str(concat_slice_operand_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时切片结果拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                concat_slice_operand_artifact,
                require_integer=False,
                require_runtime_input=True,
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_SLICE_OPERAND_CASES:
            executed = subprocess.run(
                [str(concat_slice_operand_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout[:256] + executed.stderr)
                print(f"运行时切片结果拼接 {label} 不符", file=sys.stderr)
                return 1

        nested_concat_source = ROOT / "examples" / "p1b-runtime-text-concat-slice-source.serica"
        nested_concat_artifact = temp / "runtime-text-concat-slice-source"
        checked = run("check", str(nested_concat_source))
        built = run("build", str(nested_concat_source), "-o", str(nested_concat_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("拼接结果切片作为拼接操作数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                nested_concat_artifact,
                require_integer=False,
                require_runtime_input=True,
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_concat_result = subprocess.run(
            [str(nested_concat_artifact)], cwd=ROOT, input="A行简Z\n0\n2\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_concat_result.returncode != 0 or nested_concat_result.stdout != "A行A\n".encode() or nested_concat_result.stderr:
            sys.stderr.buffer.write(nested_concat_result.stdout[:256] + nested_concat_result.stderr)
            print("两个运行时拼接源切片的别名写回结果不符", file=sys.stderr)
            return 1

        nested_text_source = ROOT / "examples" / "p1b-runtime-text-concat-nested.serica"
        nested_text_artifact = temp / "runtime-text-concat-nested"
        checked = run("check", str(nested_text_source))
        built = run("build", str(nested_text_source), "-o", str(nested_text_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("两层运行时文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_artifact, require_integer=False, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_result = subprocess.run(
            [str(nested_text_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_result.returncode != 0 or nested_text_result.stdout != "行简尾！\n".encode() or nested_text_result.stderr:
            sys.stderr.buffer.write(nested_text_result.stdout[:256] + nested_text_result.stderr)
            print("两层运行时文本拼接输出不符", file=sys.stderr)
            return 1
        nested_text_overflow = subprocess.run(
            [str(nested_text_artifact)], cwd=ROOT, input=("x" * 64 + "\n").encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_overflow.returncode == 0 or nested_text_overflow.stdout or nested_text_overflow.stderr:
            sys.stderr.buffer.write(nested_text_overflow.stdout[:256] + nested_text_overflow.stderr)
            print("嵌套文本拼接中间结果超过 63 字节时未安全拒绝", file=sys.stderr)
            return 1

        nested_text_right_source = temp / "runtime-text-concat-nested-right.serica"
        nested_text_right_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=textConcat（『前』，textConcat（『中』，文本））】。\n",
            encoding="utf-8",
        )
        nested_text_right_artifact = temp / "runtime-text-concat-nested-right"
        checked = run("check", str(nested_text_right_source))
        built = run("build", str(nested_text_right_source), "-o", str(nested_text_right_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("右侧两层运行时文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_right_artifact, require_integer=False, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_right_run = subprocess.run(
            [str(nested_text_right_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_right_run.returncode != 0 or nested_text_right_run.stdout != "前中行简\n".encode() or nested_text_right_run.stderr:
            sys.stderr.buffer.write(nested_text_right_run.stdout[:256] + nested_text_right_run.stderr)
            print("右侧两层运行时文本拼接输出不符", file=sys.stderr)
            return 1

        nested_text_length_source = temp / "runtime-text-concat-nested-length.serica"
        nested_text_length_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），"
            "输出=textLength（textConcat（textConcat（文本，『尾』），『！』））】。\n",
            encoding="utf-8",
        )
        nested_text_length_artifact = temp / "runtime-text-concat-nested-length"
        checked = run("check", str(nested_text_length_source))
        built = run("build", str(nested_text_length_source), "-o", str(nested_text_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("嵌套文本拼接的码位长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_length_artifact, require_integer=True, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_length_run = subprocess.run(
            [str(nested_text_length_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_length_run.returncode != 0 or nested_text_length_run.stdout != b"4\n" or nested_text_length_run.stderr:
            sys.stderr.buffer.write(nested_text_length_run.stdout[:256] + nested_text_length_run.stderr)
            print("嵌套文本拼接的码位长度结果不符", file=sys.stderr)
            return 1

        nested_text_three_source = temp / "runtime-text-concat-three-levels.serica"
        nested_text_three_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），"
            "输出=textConcat（textConcat（textConcat（文本，『尾』），『！』），『终』）】。\n",
            encoding="utf-8",
        )
        nested_text_three_artifact = temp / "runtime-text-concat-three-levels"
        checked = run("check", str(nested_text_three_source))
        built = run("build", str(nested_text_three_source), "-o", str(nested_text_three_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("三层运行时文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_three_artifact, require_integer=False, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_three_run = subprocess.run(
            [str(nested_text_three_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_three_run.returncode != 0 or nested_text_three_run.stdout != "行简尾！终\n".encode() or nested_text_three_run.stderr:
            sys.stderr.buffer.write(nested_text_three_run.stdout[:256] + nested_text_three_run.stderr)
            print("三层运行时文本拼接输出不符", file=sys.stderr)
            return 1

        nested_text_four_source = temp / "runtime-text-concat-four-levels.serica"
        nested_text_four_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），"
            "输出=textConcat（textConcat（textConcat（textConcat（文本，『甲』），『乙』），『丙』），『丁』）】。\n",
            encoding="utf-8",
        )
        nested_text_four_artifact = temp / "runtime-text-concat-four-levels"
        checked = run("check", str(nested_text_four_source))
        built = run("build", str(nested_text_four_source), "-o", str(nested_text_four_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("四层运行时文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_four_artifact, require_integer=False, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_four_run = subprocess.run(
            [str(nested_text_four_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_four_run.returncode != 0 or nested_text_four_run.stdout != "行简甲乙丙丁\n".encode() or nested_text_four_run.stderr:
            sys.stderr.buffer.write(nested_text_four_run.stdout[:256] + nested_text_four_run.stderr)
            print("四层运行时文本拼接输出不符", file=sys.stderr)
            return 1

        nested_text_five_source = ROOT / "examples" / "p1b-runtime-text-concat-five-levels.serica"
        nested_text_five_artifact = temp / "runtime-text-concat-five-levels"
        checked = run("check", str(nested_text_five_source))
        built = run("build", str(nested_text_five_source), "-o", str(nested_text_five_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("五层运行时文本拼接无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_five_artifact, require_integer=False, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_five_run = subprocess.run(
            [str(nested_text_five_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_five_run.returncode != 0 or nested_text_five_run.stdout != "行简甲乙丙丁戊\n".encode() or nested_text_five_run.stderr:
            sys.stderr.buffer.write(nested_text_five_run.stdout[:256] + nested_text_five_run.stderr)
            print("五层运行时文本拼接输出不符", file=sys.stderr)
            return 1

        nested_text_five_compare_source = temp / "runtime-text-concat-five-levels-compare.serica"
        nested_text_five_compare_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（equals（"
            "textConcat（textConcat（textConcat（textConcat（textConcat（文本，『甲』），『乙』），『丙』），『丁』），『戊』），"
            "『行简甲乙丙丁戊』），1，0）】。\n",
            encoding="utf-8",
        )
        nested_text_five_compare_artifact = temp / "runtime-text-concat-five-levels-compare"
        checked = run("check", str(nested_text_five_compare_source))
        built = run("build", str(nested_text_five_compare_source), "-o", str(nested_text_five_compare_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("五层运行时文本比较无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_five_compare_artifact, require_integer=True, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_five_compare_run = subprocess.run(
            [str(nested_text_five_compare_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_five_compare_run.returncode != 0 or nested_text_five_compare_run.stdout != b"1\n" or nested_text_five_compare_run.stderr:
            sys.stderr.buffer.write(nested_text_five_compare_run.stdout[:256] + nested_text_five_compare_run.stderr)
            print("五层运行时文本比较结果不符", file=sys.stderr)
            return 1

        nested_text_five_order_source = temp / "runtime-text-concat-five-levels-order.serica"
        nested_text_five_order_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=ifElse（"
            "textConcat（textConcat（textConcat（textConcat（textConcat（文本，『甲』），『乙』），『丙』），『丁』），『戊』）"
            ">=『行简甲乙丙丁戊』，1，0）】。\n",
            encoding="utf-8",
        )
        nested_text_five_order_artifact = temp / "runtime-text-concat-five-levels-order"
        checked = run("check", str(nested_text_five_order_source))
        built = run("build", str(nested_text_five_order_source), "-o", str(nested_text_five_order_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("五层运行时文本排序比较无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_five_order_artifact, require_integer=True, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_five_order_run = subprocess.run(
            [str(nested_text_five_order_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_five_order_run.returncode != 0 or nested_text_five_order_run.stdout != b"1\n" or nested_text_five_order_run.stderr:
            sys.stderr.buffer.write(nested_text_five_order_run.stdout[:256] + nested_text_five_order_run.stderr)
            print("五层运行时文本排序比较结果不符", file=sys.stderr)
            return 1

        nested_text_four_write_source = temp / "runtime-text-concat-four-levels-writeback.serica"
        nested_text_four_write_source.write_text(
            "变量 文本=readText（）。\n"
            "赋值 文本=textConcat（textConcat（textConcat（textConcat（textConcat（文本，『甲』），『乙』），『丙』），『丁』），『戊』）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=文本】。\n",
            encoding="utf-8",
        )
        nested_text_four_write_artifact = temp / "runtime-text-concat-four-levels-writeback"
        checked = run("check", str(nested_text_four_write_source))
        built = run("build", str(nested_text_four_write_source), "-o", str(nested_text_four_write_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("四层运行时文本拼接写回无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(nested_text_four_write_artifact, require_integer=False, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        nested_text_four_write_run = subprocess.run(
            [str(nested_text_four_write_artifact)], cwd=ROOT, input="行简\n".encode(), capture_output=True, check=False, timeout=5
        )
        if nested_text_four_write_run.returncode != 0 or nested_text_four_write_run.stdout != "行简甲乙丙丁戊\n".encode() or nested_text_four_write_run.stderr:
            sys.stderr.buffer.write(nested_text_four_write_run.stdout[:256] + nested_text_four_write_run.stderr)
            print("四层运行时文本拼接写回输出不符", file=sys.stderr)
            return 1

        slice_length_source = temp / "runtime-text-slice-length-large.serica"
        slice_length_source.write_text(
            "变量 文本=readText（）。\n程序=【输出=textLength（substring（文本，0，63））】。\n", encoding="utf-8"
        )
        slice_length_artifact = temp / "runtime-text-slice-length-large"
        checked = run("check", str(slice_length_source))
        built = run("build", str(slice_length_source), "-o", str(slice_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("长 UTF-8 切片码位长度无法检查或构建", file=sys.stderr)
            return 1
        slice_length = subprocess.run(
            [str(slice_length_artifact)], cwd=ROOT, input=("𠮷" * 63 + "\n").encode("utf-8"), capture_output=True, check=False, timeout=5
        )
        if slice_length.returncode != 0 or slice_length.stdout != b"63\n" or slice_length.stderr:
            sys.stderr.buffer.write(slice_length.stdout + slice_length.stderr)
            print("63 个补充平面码位切片长度不符", file=sys.stderr)
            return 1

        concat_slice_source = ROOT / "examples" / "p1b-runtime-text-concat-slice.serica"
        concat_slice_artifact = temp / "runtime-text-concat-slice"
        checked = run("check", str(concat_slice_source))
        built = run("build", str(concat_slice_source), "-o", str(concat_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时 textConcat 结果切片无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                concat_slice_artifact,
                require_integer=False,
                require_runtime_input=True,
                required_instructions=("ldrb\tw28, [x27]", "cmp\tx7, x6", "b.lt"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_SLICE_CASES:
            executed = subprocess.run(
                [str(concat_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"{label} 的拼接后码位切片输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_CONCAT_SLICE_FAILURES:
            failed = subprocess.run(
                [str(concat_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"{label} 未以无输出非零退出", file=sys.stderr)
                return 1

        concat_slice_length_source = ROOT / "examples" / "p1b-runtime-text-concat-slice-length.serica"
        concat_slice_length_artifact = temp / "runtime-text-concat-slice-length"
        checked = run("check", str(concat_slice_length_source))
        built = run("build", str(concat_slice_length_source), "-o", str(concat_slice_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("拼接结果切片的运行时码位长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                concat_slice_length_artifact,
                require_runtime_input=True,
                required_instructions=("ldrb\tw26, [x25]", "cmp\tx26, #0x80", "cmp\tx26, #0xc0"),
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_CONCAT_SLICE_LENGTH_CASES:
            executed = subprocess.run(
                [str(concat_slice_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"{label} 的拼接结果切片码位长度输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_CONCAT_SLICE_FAILURES:
            failed = subprocess.run(
                [str(concat_slice_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"{label} 的拼接切片长度路径未以无输出非零退出", file=sys.stderr)
                return 1

        text_array_source = ROOT / "examples" / "p1b-runtime-text-array.serica"
        text_array_artifact = temp / "runtime-text-array"
        checked = run("check", str(text_array_source))
        built = run("build", str(text_array_source), "-o", str(text_array_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态文本数组无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                text_array_artifact, require_integer=False, require_runtime_input=True
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in [
            ("text array index zero and alias update", "行简\n0\n".encode(), "首尾\n".encode()),
            ("text array index one and alias update", "行简\n1\n".encode(), "首尾\n".encode()),
            ("text array push retains runtime UTF-8 text", "行简\n2\n".encode(), "行简尾\n".encode()),
        ]:
            executed = subprocess.run(
                [str(text_array_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"{label} 输出不符：{executed.stdout!r}", file=sys.stderr)
                return 1
        for label, supplied in [("text array index above length", "行简\n3\n".encode()), ("text array negative index", "行简\n-1\n".encode())]:
            failed = subprocess.run(
                [str(text_array_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"{label} 未以无输出非零退出", file=sys.stderr)
                return 1

        text_array_length_source = ROOT / "examples" / "p1b-runtime-text-array-length.serica"
        text_array_length_artifact = temp / "runtime-text-array-length"
        checked = run("check", str(text_array_length_source))
        built = run("build", str(text_array_length_source), "-o", str(text_array_length_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态文本数组码位长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(text_array_length_artifact, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for index, expected in [(0, b"1\n"), (1, b"2\n")]:
            executed = subprocess.run(
                [str(text_array_length_artifact)], cwd=ROOT, input=f"{index}\n".encode(), capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"动态文本数组索引 {index} 的 Unicode 码位长度不符", file=sys.stderr)
                return 1
        for label, supplied in [("text array length index above length", b"2\n"), ("text array length negative index", b"-1\n")]:
            failed = subprocess.run(
                [str(text_array_length_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"{label} 未以无输出非零退出", file=sys.stderr)
                return 1

        text_array_get_source = ROOT / "examples" / "p1b-runtime-text-array-get.serica"
        text_array_get_artifact = temp / "runtime-text-array-get"
        checked = run("check", str(text_array_get_source))
        built = run("build", str(text_array_get_source), "-o", str(text_array_get_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时输入写入的文本数组无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                text_array_get_artifact, require_integer=False, require_runtime_input=True
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for index, expected in [(0, "首\n".encode()), (1, "尾\n".encode()), (2, "行简\n".encode())]:
            executed = subprocess.run(
                [str(text_array_get_artifact)], cwd=ROOT, input=f"行简\n{index}\n".encode(), capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时文本数组根输出索引 {index} 不符", file=sys.stderr)
                return 1
        for label, supplied in [("text array output index above length", b"\n3\n"), ("text array output negative index", b"\n-1\n")]:
            failed = subprocess.run(
                [str(text_array_get_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"{label} 未以无输出非零退出", file=sys.stderr)
                return 1

        text_array_count_source = ROOT / "examples" / "p1b-runtime-text-array-count.serica"
        text_array_count_artifact = temp / "runtime-text-array-count"
        checked = run("check", str(text_array_count_source))
        built = run("build", str(text_array_count_source), "-o", str(text_array_count_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态文本数组长度无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(text_array_count_artifact)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        text_array_count = subprocess.run(
            [str(text_array_count_artifact)], cwd=ROOT, capture_output=True, check=False, timeout=5
        )
        if text_array_count.returncode != 0 or text_array_count.stdout != b"2\n" or text_array_count.stderr:
            sys.stderr.buffer.write(text_array_count.stdout + text_array_count.stderr)
            print("动态文本数组 arrayPush / arrayLength 结果不符", file=sys.stderr)
            return 1

        text_array_slice_source = ROOT / "examples" / "p1b-runtime-text-array-slice.serica"
        text_array_slice_artifact = temp / "runtime-text-array-slice"
        checked = run("check", str(text_array_slice_source))
        built = run("build", str(text_array_slice_source), "-o", str(text_array_slice_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("运行时文本数组元素切片无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                text_array_slice_artifact, require_integer=False, require_runtime_input=True
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_ARRAY_SLICE_CASES:
            executed = subprocess.run(
                [str(text_array_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"运行时文本数组元素切片 {label} 结果不符", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_ARRAY_SLICE_FAILURES:
            failed = subprocess.run(
                [str(text_array_slice_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"运行时文本数组元素切片 {label} 未在输出前拒绝", file=sys.stderr)
                return 1

        text_array_slice_concat_source = ROOT / "examples" / "p1b-runtime-text-array-slice-concat.serica"
        text_array_slice_concat_artifact = temp / "runtime-text-array-slice-concat"
        checked = run("check", str(text_array_slice_concat_source))
        built = run("build", str(text_array_slice_concat_source), "-o", str(text_array_slice_concat_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("动态文本数组元素切片作为 textConcat 操作数无法检查或构建", file=sys.stderr)
            return 1
        try:
            verify_native_text_storage_artifact(
                text_array_slice_concat_artifact, require_integer=False, require_runtime_input=True
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        for label, supplied, expected in RUNTIME_TEXT_ARRAY_SLICE_CONCAT_CASES:
            executed = subprocess.run(
                [str(text_array_slice_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if executed.returncode != 0 or executed.stdout != expected or executed.stderr:
                sys.stderr.buffer.write(executed.stdout + executed.stderr)
                print(f"动态文本数组元素切片拼接 {label} 输出不符", file=sys.stderr)
                return 1
        for label, supplied in RUNTIME_TEXT_ARRAY_SLICE_CONCAT_FAILURES:
            failed = subprocess.run(
                [str(text_array_slice_concat_artifact)], cwd=ROOT, input=supplied, capture_output=True, check=False, timeout=5
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.buffer.write(failed.stdout + failed.stderr)
                print(f"动态文本数组元素切片拼接 {label} 未在输出前拒绝", file=sys.stderr)
                return 1

        for index, (label, source_text, supplied) in enumerate(DYNAMIC_FAILURES):
            source = temp / f"dynamic-failure-{index}.serica"
            source.write_text(source_text, encoding="utf-8")
            checked = run("check", str(source))
            artifact = temp / f"dynamic-failure-{index}"
            built = run("build", str(source), "-o", str(artifact))
            if checked.returncode != 0 or built.returncode != 0:
                sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
                print(f"{label} 应生成运行时错误路径，但编译失败", file=sys.stderr)
                return 1
            failed = subprocess.run(
                [str(artifact)],
                cwd=ROOT,
                input=supplied,
                text=True,
                capture_output=True,
                check=False,
            )
            if failed.returncode == 0 or failed.stdout or failed.stderr:
                sys.stderr.write(failed.stdout + failed.stderr)
                print(f"{label} 未以非零退出且保持无部分输出", file=sys.stderr)
                return 1

        invalid = temp / "division-zero.serica"
        invalid.write_text("程序=【输出=1/0】。\n", encoding="utf-8")
        rejected = run("check", str(invalid))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3024：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("整数除零没有在构建前被拒绝", file=sys.stderr)
            return 1

        modulo_zero = temp / "modulo-zero.serica"
        modulo_zero.write_text("程序=【输出=7%0】。\n", encoding="utf-8")
        rejected = run("check", str(modulo_zero))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3024：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("整数余数除零没有在构建前被拒绝", file=sys.stderr)
            return 1

        capability_cases = [
            ("capability-missing-field", "程序=【输出=1】。\n", "S-4101"),
            ("capability-missing-stdout", "程序=【能力=array（『stdin』），输出=1】。\n", "S-4102"),
            ("capability-missing-stdin-output", "程序=【能力=array（『stdout』），输出=readInteger（）】。\n", "S-4102"),
            (
                "capability-missing-stdin-text-concat",
                "变量 文本=textConcat（substring（readText（），0，1），『尾』）。\n"
                "程序=【能力=array（『stdout』），输出=文本】。\n",
                "S-4102",
            ),
            (
                "capability-missing-stdin-array-slice-concat",
                "变量 文本数组=array（『A行简Z』）。\n变量 索引=readInteger（）。\n"
                "变量 文本=textConcat（substring（arrayGet（文本数组，索引），1，3），『尾』）。\n"
                "程序=【能力=array（『stdout』），输出=文本】。\n",
                "S-4102",
            ),
            (
                "capability-missing-array-write",
                "变量 数组=array（1）。赋值 数组=arrayPush（数组，2）。\n"
                "程序=【能力=array（『stdout』），输出=arrayLength（数组）】。\n",
                "S-4102",
            ),
            (
                "capability-missing-map-write",
                "变量 表=映射（）。赋值 表=映射置（表，1，2）。\n"
                "程序=【能力=array（『stdout』），输出=映射长（表）】。\n",
                "S-4102",
            ),
            (
                "capability-missing-arguments",
                "程序=【能力=array（『stdout』），输出=argumentCount（）】。\n",
                "S-4102",
            ),
            (
                "capability-missing-stdin-variable",
                "变量 输入=readInteger（）。\n程序=【能力=array（『stdout』），输出=输入】。\n",
                "S-4102",
            ),
            (
                "capability-missing-stdin-condition",
                "变量 结果=0。若（readInteger（）>0）【赋值 结果=1。】。\n"
                "程序=【能力=array（『stdout』），输出=结果】。\n",
                "S-4102",
            ),
            (
                "capability-missing-stdin-loop",
                "变量 结果=0。凡（readInteger（）>0）【赋值 结果=结果+1。】。\n"
                "程序=【能力=array（『stdout』），输出=结果】。\n",
                "S-4102",
            ),
            (
                "capability-unknown",
                "程序=【能力=array（『stdout』，『filesystem.execute』），输出=1】。\n",
                "S-4103",
            ),
            (
                "capability-duplicate",
                "程序=【能力=array（『stdout』，『stdout』），输出=1】。\n",
                "S-4104",
            ),
            (
                "capability-duplicate-field",
                "程序=【能力=array（『stdout』），能力=array（『stdout』），输出=1】。\n",
                "S-4104",
            ),
            (
                "capability-wrong-type",
                "程序=【能力=array（1），输出=1】。\n",
                "S-4105",
            ),
        ]
        for name, source_text, expected_code in capability_cases:
            capability_source = temp / f"{name}.serica"
            capability_source.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = (
                    run_unmodified(
                        command,
                        str(capability_source),
                        "-o",
                        str(temp / f"{name}-{command}"),
                    )
                    if command == "build"
                    else run_unmodified(command, str(capability_source))
                )
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(f"能力检查反例 {name} 的 {command} 未返回 {expected_code}", file=sys.stderr)
                    return 1

        loop_control_cases = [
            ("break-outside-loop", "跳出。\n程序=【能力=array（『stdout』），输出=1】。\n", "S-3067"),
            ("continue-outside-loop", "继续。\n程序=【能力=array（『stdout』），输出=1】。\n", "S-3067"),
        ]
        for name, source_text, expected_code in loop_control_cases:
            loop_source = temp / f"{name}.serica"
            loop_source.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = (
                    run_unmodified(command, str(loop_source), "-o", str(temp / f"{name}-{command}"))
                    if command == "build"
                    else run_unmodified(command, str(loop_source))
                )
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(f"循环控制反例 {name} 的 {command} 未返回 {expected_code}", file=sys.stderr)
                    return 1
        missing_stdin_build = temp / "capability-missing-stdin-build.serica"
        missing_stdin_build.write_text(
            "变量 文本=textConcat（substring（readText（），0，1），『尾』）。\n"
            "程序=【能力=array（『stdout』），输出=文本】。\n",
            encoding="utf-8",
        )
        rejected = run_unmodified("build", str(missing_stdin_build), "-o", str(temp / "unauthorized"))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-4102：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("build 接受了未声明 stdin 的程序", file=sys.stderr)
            return 1

        function_budget_source = "函数 倍增0（数值）=数值+数值。\n"
        for index in range(1, 17):
            previous = f"倍增{index - 1}"
            function_budget_source += (
                f"函数 倍增{index}（数值）={previous}（数值）+{previous}（数值）。\n"
            )
        function_budget_source += (
            "程序=【能力=array（『stdout』），输出=倍增16（1）】。\n"
        )

        function_cases = [
            (
                "function-duplicate-name",
                "函数 计算（数值）=数值+1。\n函数 计算（数值）=数值+2。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3060",
            ),
            (
                "function-duplicate-parameter",
                "函数 计算（数值，数值）=数值+1。\n"
                "程序=【能力=array（『stdout』），输出=计算（1，2）】。\n",
                "S-3060",
            ),
            (
                "function-arity",
                "函数 计算（数值）=数值+1。\n"
                "程序=【能力=array（『stdout』），输出=计算（）】。\n",
                "S-3061",
            ),
            ("function-expansion-node-budget", function_budget_source, "S-3063"),
            (
                "function-direct-recursion",
                "函数 计算（文本）=计算（文本）。\n"
                "程序=【能力=array（『stdout』），输出=计算（『文本』）】。\n",
                "S-3062",
            ),
            (
                "function-indirect-recursion",
                "函数 甲（文本）=乙（文本）。\n函数 乙（文本）=甲（文本）。\n"
                "程序=【能力=array（『stdout』），输出=甲（『文本』）】。\n",
                "S-3062",
            ),
            (
                "runtime-result-function-cannot-implicitly-enter-integer-output",
                "函数 倒数（数值）=ifElse（数值==0，成功（0），倒数（数值-1））。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=倒数（readInteger（））】。\n",
                "S-3023",
            ),
            (
                "function-unknown-call",
                "函数 计算（数值）=不存在（数值）。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3064",
            ),
            (
                "function-unused-parameter",
                "函数 常量值（数值）=42。\n"
                "程序=【能力=array（『stdout』），输出=常量值（1）】。\n",
                "S-3065",
            ),
            (
                "block-function-missing-return",
                "函数 计算（数值）【变量 加倍=数值*2。】。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3068",
            ),
            (
                "block-function-duplicate-local",
                "函数 计算（数值）【变量 局部=1。变量 局部=2。返回 局部。】。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3068",
            ),
            (
                "block-function-unknown-local-assignment",
                "函数 计算（数值）【赋值 局部=2。返回 局部。】。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3068",
            ),
            (
                "block-function-forward-local",
                "函数 计算（数值）【变量 甲=乙+1。变量 乙=数值。返回 甲。】。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3068",
            ),
            (
                "block-function-branch-scope-escape",
                "函数 计算（数值）【若（数值>0）【变量 局部=1。返回 局部。】。返回 局部。】。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3068",
            ),
            (
                "block-function-unreachable-after-return",
                "函数 计算（数值）【返回 数值。变量 局部=1。】。\n"
                "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
                "S-3068",
            ),
            (
                "block-function-effectful-local",
                "函数 计算（）能力=array（『stdin』）【变量 输入=readInteger（）。返回 输入+1。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（）】。\n",
                "S-3068",
            ),
            (
                "block-function-hidden-input",
                "函数 读取（）【返回 readInteger（）。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "argument-count-arity",
                "程序=【能力=array（『stdout』，『args』），输出=argumentCount（1）】。\n",
                "S-3069",
            ),
            (
                "argument-text-negative-index",
                "程序=【能力=array（『stdout』，『args』），输出=argumentText（-1）】。\n",
                "S-3069",
            ),
            (
                "argument-text-noninteger-index",
                "程序=【能力=array（『stdout』，『args』），输出=argumentText（『1』）】。\n",
                "S-3069",
            ),
            (
                "function-hidden-arguments",
                "函数 读取（）=argumentText（1）。\n"
                "程序=【能力=array（『stdout』，『args』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-arguments-missing-root-capability",
                "函数 读取（）能力=array（『args』）=argumentText（1）。\n"
                "程序=【能力=array（『stdout』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "file-read-arity",
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=readFile（）】。\n",
                "S-3070",
            ),
            (
                "file-read-dynamic-path",
                "程序=【能力=array（『stdout』，『stdin』，『filesystem.read』），输出=readFile（readText（））】。\n",
                "S-3070",
            ),
            (
                "function-hidden-file-read",
                "函数 读取（）=readFile（『a.txt』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.read』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-file-read-missing-root-capability",
                "函数 读取（）能力=array（『filesystem.read』）=readFile（『a.txt』）。\n"
                "程序=【能力=array（『stdout』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-hidden-input",
                "函数 读取（）=readInteger（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-hidden-text-input",
                "函数 读取（）=readText（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-effect-missing-propagation",
                "函数 读取（）能力=array（『stdin』）=readInteger（）。\n"
                "函数 代理（）=读取（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=代理（）】。\n",
                "S-4106",
            ),
            (
                "function-effect-missing-root-capability",
                "函数 读取（）能力=array（『stdin』）=readInteger（）。\n"
                "程序=【能力=array（『stdout』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-effect-unknown",
                "函数 读取（）能力=array（『filesystem.execute』）=readInteger（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
                "S-4103",
            ),
            (
                "function-effect-duplicate",
                "函数 读取（）能力=array（『stdin』，『stdin』）=readInteger（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
                "S-4104",
            ),
            (
                "function-effect-wrong-type",
                "函数 读取（）能力=『stdin』=readInteger（）。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
                "S-4105",
            ),
            (
                "function-hidden-array-push",
                "函数 追加（数组）能力=array（『stdin』）=arrayPush（数组，2）。\n"
                "程序=【能力=array（『stdout』），输出=1】。\n",
                "S-4106",
            ),
            (
                "function-hidden-array-set",
                "函数 替换（数组）能力=array（『stdin』）=arraySet（数组，0，2）。\n"
                "程序=【能力=array（『stdout』），输出=1】。\n",
                "S-4106",
            ),
            (
                "function-array-write-missing-root-capability",
                "函数 追加（数组）能力=array（『array.write』）=arrayPush（数组，2）。\n"
                "变量 数组=array（1）。赋值 数组=追加（数组）。\n"
                "程序=【能力=array（『stdout』），输出=arrayGet（数组，1）】。\n",
                "S-4106",
            ),
            (
                "function-map-write-missing-root-capability",
                "函数 写入（表）=映射置（表，1，2）。\n"
                "变量 表=映射（）。赋值 表=写入（表）。\n"
                "程序=【能力=array（『stdout』），输出=映射长（表）】。\n",
                "S-4106",
            ),
            (
                "function-map-write-explicit-root-capability",
                "函数 写入（表）能力=array（『map.write』）=映射置（表，1，2）。\n"
                "函数 代理（表）=写入（表）。\n变量 表=映射（）。赋值 表=代理（表）。\n"
                "程序=【能力=array（『stdout』，『map.write』），输出=映射长（表）】。\n",
                "S-4106",
            ),
            (
                "function-map-write-missing-propagation",
                "函数 写入（表）能力=array（『map.write』）=映射置（表，1，2）。\n"
                "函数 代理（表）能力=array（『map.write』）=写入（表）。\n"
                "变量 表=映射（）。赋值 表=代理（表）。\n"
                "程序=【能力=array（『stdout』），输出=映射长（表）】。\n",
                "S-4106",
            ),
            (
                "function-array-write-missing-propagation",
                "函数 追加（数组）能力=array（『array.write』）=arrayPush（数组，2）。\n"
                "函数 代理（数组）=追加（数组）。\n"
                "变量 数组=array（1）。赋值 数组=代理（数组）。\n"
                "程序=【能力=array（『stdout』，『array.write』），输出=arrayGet（数组，1）】。\n",
                "S-4106",
            ),
            (
                "function-effectful-argument-used-twice",
                "函数 倍增（数值）能力=array（『stdin』）=数值+数值。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=倍增（readInteger（））】。\n",
                "S-4106",
            ),
            (
                "function-effectful-user-function-argument",
                "函数 读取（）能力=array（『stdin』）=readInteger（）。\n"
                "函数 倍增（数值）能力=array（『stdin』）=数值+数值。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=倍增（读取（））】。\n",
                "S-4106",
            ),
            (
                "runtime-block-loop-unknown-assignment",
                "函数 计算（数值）【凡（数值>0）【赋值 结果=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-non-boolean-condition",
                "函数 计算（数值）【变量 当前=数值。凡（当前）【赋值 当前=当前-1。】。返回 当前。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-static-true-condition",
                "函数 计算（数值）【凡（1==1）【赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-missing-return",
                "函数 计算（数值）【凡（数值>0）【赋值 数值=数值-1。】。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-hidden-input",
                "函数 计算（数值）【凡（数值>0）【赋值 数值=readInteger（）。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-4106",
            ),
            (
                "runtime-block-loop-text-local",
                "函数 计算（数值）【变量 文本=『占位』。凡（数值>0）【赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-break-outside-loop",
                "函数 计算（数值）【跳出。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-continue-outside-loop",
                "函数 计算（数值）【继续。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-local-duplicate",
                "函数 计算（数值）【变量 增量=1。凡（数值>0）【变量 增量=数值-1。赋值 数值=增量。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-local-scope-escape",
                "函数 计算（数值）【凡（数值>0）【变量 增量=数值-1。赋值 数值=增量。】。返回 增量。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3022",
            ),
            (
                "runtime-block-loop-text-local-inside",
                "函数 计算（数值）【凡（数值>0）【变量 增量=『文本』。赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-local-hidden-input",
                "函数 计算（数值）【凡（数值>0）【变量 增量=readInteger（）。赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-4106",
            ),
            (
                "runtime-block-loop-return-unreachable",
                "函数 计算（数值）【凡（数值>0）【返回 数值。赋值 数值=0。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-return-text",
                "函数 计算（数值）【凡（数值>0）【返回 『文本』。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-loop-return-hidden-input",
                "函数 计算（数值）【凡（数值>0）【返回 readInteger（）。】。返回 0。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-4106",
            ),
            (
                "runtime-block-top-level-if-non-boolean",
                "函数 计算（数值）【若（数值）【赋值 数值=0。】。凡（数值>0）【赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-top-level-if-break",
                "函数 计算（数值）【若（数值>0）【跳出。】。凡（数值>0）【赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3068",
            ),
            (
                "runtime-block-top-level-branch-local-escape",
                "函数 计算（数值）【若（数值>0）【变量 局部=数值。赋值 数值=局部。】。凡（数值>0）【赋值 数值=数值-1。】。返回 局部。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-3022",
            ),
            (
                "runtime-block-top-level-if-hidden-input",
                "函数 计算（数值）【若（readInteger（）>0）【赋值 数值=0。】。凡（数值>0）【赋值 数值=数值-1。】。返回 数值。】。\n"
                "程序=【能力=array（『stdout』，『stdin』），输出=计算（readInteger（））】。\n",
                "S-4106",
            ),
        ]
        for name, source_text, expected_code in function_cases:
            function_case = temp / f"{name}.serica"
            function_case.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = run_unmodified(
                    command,
                    str(function_case),
                    "-o",
                    str(temp / f"{name}-{command}"),
                ) if command == "build" else run_unmodified(command, str(function_case))
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(
                        f"纯函数诊断反例 {name} 的 {command} 未返回 {expected_code}",
                        file=sys.stderr,
                    )
                    return 1

        file_write_diagnostics = [
            (
                "file-write-arity",
                "程序=【能力=array（『stdout』，『filesystem.write』），输出=writeFile（）】。\n",
                "S-3071",
            ),
            (
                "file-write-dynamic-path",
                "程序=【能力=array（『stdout』，『stdin』，『filesystem.write』），"
                "输出=writeFile（readText（），『x』）】。\n",
                "S-3071",
            ),
            (
                "file-write-integer-path",
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（1，『x』）】。\n",
                "S-3071",
            ),
            (
                "file-write-empty-path",
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『』，『x』）】。\n",
                "S-3071",
            ),
            (
                "file-write-non-text-content",
                "程序=【能力=array（『stdout』，『filesystem.write』），"
                "输出=writeFile（『x.txt』，1）】。\n",
                "S-3071",
            ),
            (
                "file-write-missing-root-capability",
                "变量 字节数=writeFile（『x.txt』，『x』）。\n"
                "程序=【能力=array（『stdout』），输出=字节数】。\n",
                "S-4102",
            ),
            (
                "function-hidden-file-write",
                "函数 写入（）=writeFile（『x.txt』，『x』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.write』），输出=写入（）】。\n",
                "S-4106",
            ),
            (
                "function-file-write-missing-root-capability",
                "函数 写入（）能力=array（『filesystem.write』）=writeFile（『x.txt』，『x』）。\n"
                "程序=【能力=array（『stdout』），输出=写入（）】。\n",
                "S-4106",
            ),
            (
                "function-file-write-missing-propagation",
                "函数 写入（）能力=array（『filesystem.write』）=writeFile（『x.txt』，『x』）。\n"
                "函数 代理（）=写入（）。\n"
                "程序=【能力=array（『stdout』，『filesystem.write』），输出=代理（）】。\n",
                "S-4106",
            ),
            (
                "dynamic-file-read-missing-args-root",
                "程序=【能力=array（『stdout』，『filesystem.read』），"
                "输出=readFile（argumentText（1））】。\n",
                "S-4102",
            ),
            (
                "dynamic-file-write-missing-args-root",
                "变量 字节数=writeFile（argumentText（1），『x』）。\n"
                "程序=【能力=array（『stdout』，『filesystem.write』），输出=字节数】。\n",
                "S-4102",
            ),
            (
                "function-hidden-dynamic-file-read-args-effect",
                "函数 读取（）能力=array（『filesystem.read』）=readFile（argumentText（1））。\n"
                "程序=【能力=array（『stdout』，『args』，『filesystem.read』），输出=读取（）】。\n",
                "S-4106",
            ),
            (
                "function-hidden-dynamic-file-write-args-effect",
                "函数 写入（）能力=array（『filesystem.write』）=writeFile（argumentText（1），『x』）。\n"
                "程序=【能力=array（『stdout』，『args』，『filesystem.write』），输出=写入（）】。\n",
                "S-4106",
            ),
            (
                "function-dynamic-file-read-missing-propagation",
                "函数 读取（）能力=array（『args』，『filesystem.read』）=readFile（argumentText（1））。\n"
                "函数 代理（）能力=array（『filesystem.read』）=读取（）。\n"
                "程序=【能力=array（『stdout』，『args』，『filesystem.read』），输出=代理（）】。\n",
                "S-4106",
            ),
        ]
        for name, source_text, expected_code in file_write_diagnostics:
            diagnostic_source = temp / f"{name}.serica"
            diagnostic_source.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = (
                    run_unmodified(
                        command,
                        str(diagnostic_source),
                        "-o",
                        str(temp / f"{name}-{command}"),
                    )
                    if command == "build"
                    else run_unmodified(command, str(diagnostic_source))
                )
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(
                        f"文件写入诊断反例 {name} 的 {command} 未返回 {expected_code}",
                        file=sys.stderr,
                    )
                    return 1
        function_depth = "".join(
            f"函数 层{index}（数值）=层{index + 1}（数值）。\n" for index in range(64)
        )
        function_depth += "函数 层64（数值）=数值。\n程序=【能力=array（『stdout』），输出=层0（1）】。\n"
        function_depth_source = temp / "function-expansion-depth.serica"
        function_depth_source.write_text(function_depth, encoding="utf-8")
        rejected = run_unmodified("check", str(function_depth_source))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3063：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("超过函数展开深度的调用图没有被拒绝", file=sys.stderr)
            return 1
        for name, source_text, expected_code in [
            ("overflow", "程序=【输出=9223372036854775807+1】。\n", "S-3018"),
            ("array-output", "程序=【输出=array（1）】。\n", "S-3023"),
            ("unknown-assignment", "赋值 未声明=1。\n程序=【输出=1】。\n", "S-3038"),
            ("duplicate-variable", "变量 数值=1。\n变量 数值=2。\n程序=【输出=数值】。\n", "S-3037"),
            ("forward-variable-reference", "变量 后项=前项。\n变量 前项=1。\n程序=【输出=后项】。\n", "S-3022"),
            ("variable-overflow", "变量 大值=9223372036854775807。\n变量 一=1。\n程序=【输出=大值+一】。\n", "S-3018"),
            ("variable-division-zero", "变量 数值=1。\n变量 除数=0。\n程序=【输出=数值/除数】。\n", "S-3024"),
            ("variable-multiplication-overflow", "变量 大值=9223372036854775807。\n赋值 大值=大值*2。\n程序=【输出=大值】。\n", "S-3018"),
            ("variable-in-window", "变量 数值=1。\n窗口=【标题=『窗口』，文本=『正文』】。\n", "S-3003"),
            ("negative-shift-count", "程序=【输出=1<<-1】。\n", "S-3041"),
            ("large-shift-count", "程序=【输出=1>>64】。\n", "S-3041"),
            ("left-shift-overflow", "程序=【输出=1<<63】。\n", "S-3018"),
            ("runtime-input-constant", "常量 输入=readInteger（）。\n程序=【输出=输入】。\n", "S-3042"),
            ("runtime-input-arity", "程序=【输出=readInteger（1）】。\n", "S-3043"),
            ("runtime-string-add", "程序=【输出=readInteger（）+『x』】。\n", "S-3023"),
            ("runtime-array-input", "常量 项=array（readInteger（））。\n程序=【输出=1】。\n", "S-3031"),
            ("runtime-array-index", "常量 项=array（1）。\n程序=【输出=arrayGet（项，readInteger（））】。\n", "S-3034"),
            ("equals-arity", "程序=【输出=ifElse（equals（1），1，2）】。\n", "S-3017"),
            ("equals-mismatched-types", "程序=【输出=ifElse（equals（1，『1』），1，2）】。\n", "S-3015"),
            ("ifelse-mismatched-branches", "程序=【输出=ifElse（equals（1，1），1，『x』）】。\n", "S-3023"),
            ("string-order-mixed-types", "程序=【输出=ifElse（『a』<1，1，2）】。\n", "S-3023"),
            ("runtime-text-order-wrong-type", "变量 文本=readText（）。\n若（文本<1）【】。\n程序=【输出=0】。\n", "S-3023"),
            ("runtime-text-order-slice-dynamic-ifelse", "若（substring（ifElse（equals（readText（），『x』），『ab』，『cd』），0，1）<『z』）【】。\n程序=【能力=array（『stdout』，『stdin』），输出=0】。\n", "S-3042"),
            ("forbidden-not-equal-symbol", "程序=【输出=ifElse（1!=2，1，2）】。\n", "S-1001"),
            ("integer-if-condition", "变量 数值=1。\n若（数值）【赋值 数值=2。】。\n程序=【输出=数值】。\n", "S-3015"),
            ("unknown-assignment-in-control", "若（1<2）【赋值 未声明=1。】。\n程序=【输出=1】。\n", "S-3038"),
            ("variable-declaration-in-control", "若（1<2）【变量 数值=1。】。\n程序=【输出=1】。\n", "S-3037"),
            ("logical-condition-type", "程序=【输出=ifElse（1 且 2，1，0）】。\n", "S-3015"),
            ("array-set-negative-index", "变量 数组=array（1，2）。\n赋值 数组=arraySet（数组，-1，9）。\n程序=【输出=1】。\n", "S-3032"),
            ("array-set-target-mismatch", "变量 数组=array（1，2）。\n赋值 数组=arraySet（array（1，2），0，9）。\n程序=【输出=1】。\n", "S-3033"),
            ("array-set-noninteger-value", "变量 数组=array（1，2）。\n赋值 数组=arraySet（数组，0，『x』）。\n程序=【输出=1】。\n", "S-3031"),
            ("array-push-wrong-target", "变量 第一=array（1）。\n变量 第二=array（2）。\n赋值 第二=arrayPush（第一，3）。\n程序=【输出=1】。\n", "S-3033"),
            ("array-push-noninteger", "变量 数组=array（1）。\n赋值 数组=arrayPush（数组，『x』）。\n程序=【输出=1】。\n", "S-3031"),
            ("array-push-wrong-arity", "变量 数组=array（1）。\n赋值 数组=arrayPush（数组，2，3）。\n程序=【输出=1】。\n", "S-3035"),
            ("text-array-set-type-mismatch", "变量 文本数组=array（『a』）。\n赋值 文本数组=arraySet（文本数组，0，1）。\n程序=【输出=1】。\n", "S-3031"),
            ("text-array-push-type-mismatch", "变量 文本数组=array（『a』）。\n赋值 文本数组=arrayPush（文本数组，1）。\n程序=【输出=1】。\n", "S-3031"),
            ("read-text-arity", "程序=【输出=readText（1）】。\n", "S-3044"),
            ("read-text-constant", "常量 文本=readText（）。\n程序=【输出=『x』】。\n", "S-3042"),
            ("read-text-to-integer", "变量 数值=1。\n赋值 数值=readText（）。\n程序=【输出=数值】。\n", "S-3023"),
            ("integer-to-read-text-variable", "变量 文本=readText（）。\n赋值 文本=1。\n程序=【输出=文本】。\n", "S-3023"),
            ("text-concat-arity", "变量 文本=readText（）。\n变量 结果=textConcat（文本）。\n程序=【输出=结果】。\n", "S-3044"),
            ("text-concat-integer", "变量 文本=readText（）。\n变量 结果=textConcat（文本，1）。\n程序=【输出=结果】。\n", "S-3044"),
            ("text-concat-overlong-static", "变量 文本=readText（）。\n变量 结果=textConcat（文本，『" + "x" * 64 + "』）。\n程序=【输出=结果】。\n", "S-3044"),
            ("text-concat-slice-deep-source", "变量 文本=readText（）。\n变量 结果=textConcat（substring（textConcat（substring（textConcat（文本，『a』），0，1），『b』），0，1），『c』）。\n程序=【输出=结果】。\n", "S-3013"),
            ("text-concat-to-integer", "变量 数值=1。\n赋值 数值=textConcat（数值，数值）。\n程序=【输出=数值】。\n", "S-3044"),
            ("runtime-text-equals-conditional-expression", "变量 文本=readText（）。\n若（equals（文本，ifElse（equals（1，1），文本，文本）））【】。\n程序=【输出=0】。\n", "S-3023"),
            ("runtime-text-equals-integer", "变量 文本=readText（）。\n若（equals（文本，readInteger（）））【】。\n程序=【输出=0】。\n", "S-3015"),
            ("runtime-text-not-equals-arity", "变量 文本=readText（）。\n若（notEquals（文本））【】。\n程序=【输出=0】。\n", "S-3017"),
            ("runtime-text-slice-negative", "变量 文本=readText（）。\n程序=【输出=substring（文本，-1，1）】。\n", "S-3013"),
            ("runtime-text-slice-reversed", "变量 文本=readText（）。\n程序=【输出=substring（文本，2，1）】。\n", "S-3013"),
            ("runtime-text-slice-too-large", "变量 文本=readText（）。\n程序=【输出=substring（文本，0，64）】。\n", "S-3013"),
            ("runtime-text-slice-wrong-index-type", "变量 文本=readText（）。\n变量 索引=readText（）。\n程序=【输出=substring（文本，索引，2）】。\n", "S-3013"),
        ]:
            invalid = temp / f"{name}.serica"
            invalid.write_text(source_text, encoding="utf-8")
            rejected = run("check", str(invalid))
            diagnostic = rejected.stdout + rejected.stderr
            if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                sys.stderr.write(diagnostic)
                print(f"错误输入 {name} 未返回 {expected_code}", file=sys.stderr)
                return 1

        boolean_type_failures = [
            ("boolean-to-integer", "变量 标志=1<2。赋值 标志=1。程序=【输出=0】。\n", "S-3023"),
            ("integer-to-boolean", "变量 数值=1。赋值 数值=1<2。程序=【输出=0】。\n", "S-3023"),
            ("boolean-arithmetic", "变量 标志=1<2。程序=【输出=标志+1】。\n", "S-3023"),
            ("integer-pushed-to-boolean-array", "变量 数组=array（1<2）。赋值 数组=arrayPush（数组，1）。程序=【输出=0】。\n", "S-3031"),
            ("boolean-pushed-to-integer-array", "变量 数组=array（1）。赋值 数组=arrayPush（数组，1<2）。程序=【输出=0】。\n", "S-3031"),
            ("boolean-array-arithmetic", "变量 数组=array（1<2）。程序=【输出=arrayGet（数组，0）+1】。\n", "S-3023"),
            ("integer-array-condition", "变量 数组=array（1）。程序=【输出=ifElse（arrayGet（数组，0），42，-1）】。\n", "S-3015"),
            ("boolean-integer-equality", "变量 标志=readInteger（）>0。变量 数值=readInteger（）。程序=【能力=array（『stdout』，『stdin』），输出=ifElse（equals（标志，数值），42，-1）】。\n", "S-3015"),
        ]
        for name, source_text, expected_code in boolean_type_failures:
            invalid = temp / f"{name}.serica"
            invalid.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = run(command, str(invalid), "-o", str(temp / f"{name}-{command}")) if command == "build" else run(command, str(invalid))
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(f"布尔类型边界 {name} 的 {command} 未返回 {expected_code}", file=sys.stderr)
                    return 1

        map_type_failures = [
            ("map-constructor-argument", "变量 表=映射（1）。程序=【能力=array（『stdout』），输出=0】。\n", "S-3011"),
            ("map-set-arity", "变量 表=映射（）。赋值 表=映射置（表，1）。程序=【能力=array（『stdout』，『map.write』），输出=0】。\n", "S-3011"),
            ("map-set-other-target", "变量 甲=映射（）。变量 乙=映射（）。赋值 乙=映射置（甲，1，2）。程序=【能力=array（『stdout』，『map.write』），输出=0】。\n", "S-3023"),
            ("map-get-arity", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=映射取（表，1）】。\n", "S-3011"),
            ("map-get-non-map", "变量 数值=1。程序=【能力=array（『stdout』），输出=映射取（数值，1，0）】。\n", "S-3015"),
            ("map-get-wrong-default", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=映射取（表，1，『默认』）】。\n", "S-3015"),
            ("map-has-arity", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=映射有（表）】。\n", "S-3011"),
            ("map-has-wrong-key", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=映射有（表，『键』）】。\n", "S-3015"),
            ("map-length-arity", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=映射长（表，1）】。\n", "S-3011"),
            ("map-length-non-map", "变量 数值=1。程序=【能力=array（『stdout』），输出=映射长（数值）】。\n", "S-3015"),
            ("map-direct-output", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=表】。\n", "S-3023"),
            ("map-to-integer-variable", "变量 表=映射（）。变量 数值=表。程序=【能力=array（『stdout』），输出=数值】。\n", "S-3023"),
            ("integer-to-map-assignment", "变量 表=映射（）。赋值 表=1。程序=【能力=array（『stdout』），输出=0】。\n", "S-3023"),
            ("map-as-condition", "变量 表=映射（）。若（表）【】。程序=【能力=array（『stdout』），输出=0】。\n", "S-3015"),
            ("map-integer-equality", "变量 表=映射（）。程序=【能力=array（『stdout』），输出=ifElse（equals（表，1），1，0）】。\n", "S-3015"),
        ]
        for name, source_text, expected_code in map_type_failures:
            invalid = temp / f"{name}.serica"
            invalid.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = run(command, str(invalid), "-o", str(temp / f"{name}-{command}")) if command == "build" else run(command, str(invalid))
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(f"映射类型边界 {name} 的 {command} 未返回 {expected_code}", file=sys.stderr)
                    return 1

        max_maps = "".join(f"变量 表{index}=映射（）。\n" for index in range(167))
        max_maps += "程序=【能力=array（『stdout』），输出=映射长（表166）】。\n"
        max_maps_source = temp / "map-slot-limit-ok.serica"
        max_maps_source.write_text(max_maps, encoding="utf-8")
        checked = run_unmodified("check", str(max_maps_source))
        max_maps_artifact = temp / "map-slot-limit-ok"
        built = run_unmodified("build", str(max_maps_source), "-o", str(max_maps_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("167 个三槽映射变量的栈上限内程序无法构建", file=sys.stderr)
            return 1
        max_maps_run = subprocess.run([str(max_maps_artifact)], cwd=ROOT, text=True, capture_output=True, check=False)
        if max_maps_run.returncode != 0 or max_maps_run.stdout != "0\n" or max_maps_run.stderr:
            sys.stderr.write(max_maps_run.stdout + max_maps_run.stderr)
            print("167 个映射变量的三槽边界读取失败", file=sys.stderr)
            return 1
        too_many_maps = max_maps.replace("程序=【", "变量 表167=映射（）。\n程序=【", 1)
        too_many_maps_source = temp / "map-slot-limit-too-many.serica"
        too_many_maps_source.write_text(too_many_maps, encoding="utf-8")
        rejected = run_unmodified("check", str(too_many_maps_source))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3040：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("168 个三槽映射变量没有触发栈上限诊断", file=sys.stderr)
            return 1

        result_type_failures = [
            ("result-success-no-argument", "变量 结果=成功（）。\n程序=【能力=array（『stdout』），输出=0】。\n", "S-3011"),
            ("result-failure-two-arguments", "变量 结果=失败（1，2）。\n程序=【能力=array（『stdout』），输出=0】。\n", "S-3011"),
            ("result-failure-zero-code", "变量 结果=失败（0）。\n程序=【能力=array（『stdout』），输出=错误码（结果）】。\n", "S-3013"),
            ("result-failure-negative-code", "变量 结果=失败（-1）。\n程序=【能力=array（『stdout』），输出=错误码（结果）】。\n", "S-3013"),
            ("result-failure-byte-overflow", "变量 结果=失败（256）。\n程序=【能力=array（『stdout』），输出=错误码（结果）】。\n", "S-3013"),
            ("text-result-to-integer-payload", "变量 结果=成功（『文本』）。\n赋值 结果=成功（1）。\n程序=【能力=array（『stdout』），输出=错误码（结果）】。\n", "S-3023"),
            ("integer-result-to-text-payload", "变量 结果=成功（1）。\n赋值 结果=成功（『文本』）。\n程序=【能力=array（『stdout』），输出=错误码（结果）】。\n", "S-3023"),
            ("text-result-value-arithmetic", "变量 结果=成功（『文本』）。\n程序=【能力=array（『stdout』），输出=结果值（结果）+1】。\n", "S-3023"),
            ("integer-result-text-length", "变量 结果=成功（42）。\n程序=【能力=array（『stdout』），输出=textLength（结果值（结果））】。\n", "S-3044"),
            ("integer-result-text-concat", "变量 结果=成功（42）。\n程序=【能力=array（『stdout』），输出=textConcat（结果值（结果），『！』）】。\n", "S-3044"),
            ("integer-result-text-comparison", "变量 结果=成功（42）。\n程序=【能力=array（『stdout』），输出=ifElse（equals（结果值（结果），『42』），1，0）】。\n", "S-3015"),
            ("result-status-no-argument", "变量 结果=成功（1）。\n程序=【能力=array（『stdout』），输出=是否失败（）】。\n", "S-3013"),
            ("result-value-non-reference", "程序=【能力=array（『stdout』），输出=结果值（成功（1））】。\n", "S-3013"),
            ("result-status-non-reference", "程序=【能力=array（『stdout』），输出=是否失败（成功（1））】。\n", "S-3013"),
            ("result-error-code-non-reference", "程序=【能力=array（『stdout』），输出=错误码（成功（1））】。\n", "S-3013"),
            ("result-error-wrong-variable", "变量 数值=1。\n程序=【能力=array（『stdout』），输出=错误码（数值）】。\n", "S-3015"),
            ("result-direct-output", "变量 结果=成功（42）。\n程序=【能力=array（『stdout』），输出=结果】。\n", "S-3023"),
            ("result-to-integer-variable", "变量 结果=成功（42）。\n变量 数值=结果。\n程序=【能力=array（『stdout』），输出=数值】。\n", "S-3023"),
            ("integer-to-result-variable", "变量 结果=成功（42）。\n赋值 结果=1。\n程序=【能力=array（『stdout』），输出=错误码（结果）】。\n", "S-3023"),
            ("result-boolean-equality", "变量 结果=成功（42）。\n程序=【能力=array（『stdout』），输出=ifElse（equals（结果，1），1，0）】。\n", "S-3015"),
            ("result-as-condition", "变量 结果=成功（42）。\n若（结果）【】。\n程序=【能力=array（『stdout』），输出=0】。\n", "S-3015"),
            ("result-if-else-arity", "变量 结果=ifElse（1==1，成功（1））。\n程序=【能力=array（『stdout』），输出=0】。\n", "S-3016"),
            ("result-if-else-payload-mismatch", "变量 结果=ifElse（readInteger（）==1，成功（1），成功（『文本』））。\n程序=【能力=array（『stdout』，『stdin』），输出=错误码（结果）】。\n", "S-3023"),
            ("result-if-else-mixed-branch", "变量 结果=ifElse（readInteger（）==1，成功（1），2）。\n程序=【能力=array（『stdout』，『stdin』），输出=错误码（结果）】。\n", "S-3023"),
            ("result-function-hidden-stdin", "函数 读取（开关）=ifElse（开关==1，成功（readText（）），失败（8））。\n变量 结果=读取（1）。\n程序=【能力=array（『stdout』，『stdin』），输出=结果值（结果）】。\n", "S-4106"),
            ("effectful-ordinary-text-local", "函数 读取（）能力=array（『stdin』）【\n变量 文本=readText（）。\n返回 0。\n】。\n程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n", "S-4106"),
            ("nested-function-hidden-root-stdin", "函数 读取（）能力=array（『stdin』）=readInteger（）。\n函数 三倍输入（）能力=array（『stdin』）=读取（）*3。\n程序=【能力=array（『stdout』），输出=三倍输入（）】。\n", "S-4106"),
            ("nested-function-hidden-root-array-write", "函数 追加（数组）能力=array（『array.write』）=arrayPush（数组，42）。\n函数 写入（数组）能力=array（『array.write』）=追加（数组）。\n变量 数组=array（7）。\n赋值 数组=写入（数组）。\n程序=【能力=array（『stdout』），输出=arrayLength（数组）】。\n", "S-4106"),
            ("result-to-text-variable", "变量 文本=readText（）。\n变量 结果=成功（42）。\n赋值 文本=结果。\n程序=【能力=array（『stdout』，『stdin』），输出=文本】。\n", "S-3023"),
        ]
        for name, source_text, expected_code in result_type_failures:
            invalid = temp / f"{name}.serica"
            invalid.write_text(source_text, encoding="utf-8")
            for command in ("check", "build"):
                rejected = run_unmodified(
                    command,
                    str(invalid),
                    "-o",
                    str(temp / f"{name}-{command}"),
                ) if command == "build" else run_unmodified(command, str(invalid))
                diagnostic = rejected.stdout + rejected.stderr
                if rejected.returncode == 0 or f"错误 {expected_code}：" not in diagnostic:
                    sys.stderr.write(diagnostic)
                    print(f"结果类型边界 {name} 的 {command} 未返回 {expected_code}", file=sys.stderr)
                    return 1

        max_results = "".join(f"变量 结果{index}=成功（{index}）。\n" for index in range(251))
        max_results += "程序=【能力=array（『stdout』），输出=错误码（结果250）】。\n"
        max_results_source = temp / "result-slot-limit-ok.serica"
        max_results_source.write_text(max_results, encoding="utf-8")
        checked = run_unmodified("check", str(max_results_source))
        max_results_artifact = temp / "result-slot-limit-ok"
        built = run_unmodified("build", str(max_results_source), "-o", str(max_results_artifact))
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("251 个双槽结果变量的栈上限内程序无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(max_results_artifact, require_variable_storage=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        max_results_run = subprocess.run(
            [str(max_results_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if max_results_run.returncode != 0 or max_results_run.stdout != "0\n" or max_results_run.stderr:
            sys.stderr.write(max_results_run.stdout + max_results_run.stderr)
            print("251 个结果变量的双槽边界读取失败", file=sys.stderr)
            return 1
        too_many_results = max_results.replace("结果250=", "结果251=").replace(
            "输出=错误码（结果250）", "输出=错误码（结果251）"
        )
        too_many_results = "变量 结果252=成功（252）。\n" + too_many_results
        too_many_results_source = temp / "result-slot-limit-exceeded.serica"
        too_many_results_source.write_text(too_many_results, encoding="utf-8")
        rejected = run_unmodified("check", str(too_many_results_source))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3040：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("超过 502 个双槽结果变量未被拒绝", file=sys.stderr)
            return 1

        max_text_results = "".join(
            f"变量 文本结果{index}=成功（『载荷{index}』）。\n" for index in range(125)
        )
        max_text_results += "程序=【能力=array（『stdout』），输出=错误码（文本结果124）】。\n"
        max_text_results_source = temp / "text-result-slot-limit-ok.serica"
        max_text_results_source.write_text(max_text_results, encoding="utf-8")
        max_text_results_artifact = temp / "text-result-slot-limit-ok"
        checked = run_unmodified("check", str(max_text_results_source))
        built = run_unmodified(
            "build",
            str(max_text_results_source),
            "-o",
            str(max_text_results_artifact),
        )
        if checked.returncode != 0 or built.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr + built.stdout + built.stderr)
            print("125 个四槽文本结果变量的栈上限内程序无法构建", file=sys.stderr)
            return 1
        max_text_results_run = subprocess.run(
            [str(max_text_results_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if max_text_results_run.returncode != 0 or max_text_results_run.stdout != "0\n" or max_text_results_run.stderr:
            sys.stderr.write(max_text_results_run.stdout + max_text_results_run.stderr)
            print("125 个文本结果变量的四槽边界读取失败", file=sys.stderr)
            return 1
        too_many_text_results = "变量 文本结果125=成功（『溢出』）。\n" + max_text_results
        too_many_text_results_source = temp / "text-result-slot-limit-exceeded.serica"
        too_many_text_results_source.write_text(too_many_text_results, encoding="utf-8")
        rejected = run_unmodified("check", str(too_many_text_results_source))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3040：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("超过 502 个四槽文本结果变量未被拒绝", file=sys.stderr)
            return 1

        concat_depth_source = temp / "text-concat-depth-exceeded.serica"
        concat_depth_source.write_text(
            "变量 文本=readText（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），"
            "输出=textConcat（textConcat（textConcat（textConcat（textConcat（textConcat（文本，『a』），『b』），『c』），『d』），『e』），『f』）】。\n",
            encoding="utf-8",
        )
        for command in ("check", "build"):
            rejected = (
                run(command, str(concat_depth_source), "-o", str(temp / "text-concat-depth-exceeded"))
                if command == "build"
                else run(command, str(concat_depth_source))
            )
            diagnostic = rejected.stdout + rejected.stderr
            if rejected.returncode == 0 or "错误 S-3044：" not in diagnostic:
                sys.stderr.write(diagnostic)
                print(f"六层动态拼接的 {command} 未返回 S-3044", file=sys.stderr)
                return 1

        too_many_variables = "".join(f"变量 数值{index}=0。\n" for index in range(503))
        too_many_variables += "程序=【输出=0】。\n"
        invalid.write_text(too_many_variables, encoding="utf-8")
        rejected = run("check", str(invalid))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3040：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("超过原生栈帧上限的变量数未被拒绝", file=sys.stderr)
            return 1

        too_many_text_variables = "".join(f"变量 文本{index}=readText（）。\n" for index in range(56))
        too_many_text_variables += "程序=【输出=1】。\n"
        invalid.write_text(too_many_text_variables, encoding="utf-8")
        rejected = run("check", str(invalid))
        diagnostic = rejected.stdout + rejected.stderr
        if rejected.returncode == 0 or "错误 S-3040：" not in diagnostic:
            sys.stderr.write(diagnostic)
            print("运行时文本变量未计入原生栈帧上限", file=sys.stderr)
            return 1

    print(
        f"P1b 原生输出子门通过：{len(CASES) + len(DYNAMIC_CASES) + len(MAP_CASES) + len(RESULT_FUNCTION_CASES) + len(BLOCK_FUNCTION_CASES) + len(RUNTIME_BLOCK_LOOP_CASES) + len(RUNTIME_TEXT_PARAMETER_CASES) + len(RUNTIME_FUNCTION_ABI_CASES) + len(RUNTIME_RESULT_ABI_CASES) + 6} 个 arm64 整数形态产物和 1 个静态 UTF-8 文本产物执行，其中 {len(MAP_CASES)} 个覆盖整数键值哈希映射；另有 5 个静态文件输入成功产物（另有 3 个无输出失败）、1 个参数文本动态路径读取产物（另有 {dynamic_file_path_failure_count // 2} 个无输出失败）、{len(file_variable_cases)} 个文件文本变量成功产物（另有 {len(file_variable_failures)} 个无输出失败）、{len(file_write_cases)} 个文件写入成功产物（另有 {len(file_write_failures) + 1} 个无输出失败）和 1 个参数文本动态路径写入产物（另有 {dynamic_file_path_failure_count - dynamic_file_path_failure_count // 2} 个无输出失败）、{len(TEXT_RESULT_CASES)} 个静态文本结果产物、{len(DYNAMIC_TEXT_RESULT_CASES)} 个运行时文本结果产物、{len(RESULT_FUNCTION_CASES)} 个函数结果边界产物、{len(BLOCK_FUNCTION_CASES)} 个块体函数局部作用域产物、{len(RUNTIME_BLOCK_LOOP_CASES)} 个运行时整数块循环函数产物、{len(RUNTIME_TEXT_PARAMETER_CASES)} 个运行时文本参数比较/长度/本地副本桥产物、{len(OWNED_TEXT_RETURN_CASES)} 个文本所有权返回产物、{len(OWNED_TEXT_RETURN_STORAGE_CASES)} 个文本返回调用方存储产物、{len(RUNTIME_FUNCTION_ABI_CASES)} 个递归运行时函数调用帧产物、{len(RUNTIME_RESULT_ABI_CASES)} 个递归整数结果 ABI 产物、{len(RUNTIME_TEXT_RESULT_ABI_CASES)} 个递归编译期文本结果 ABI 产物、1 个直接输入、1 个运行时整数纯函数、1 个复合实参运行时整数纯函数、1 个运行时文本纯函数、1 个整数 stdin 效果函数、1 个文本 stdin 效果函数和 1 个嵌套 array.write 效果函数产物、4 个文本长度、2 个文本变量、4 个拼接、1 个 readText 拼接、1 个 readText 切片拼接、1 个文本数组元素切片、1 个数组元素切片拼接、10 个嵌套文本拼接、6 个文本相等比较（含五层拼接比较）、3 个文本排序比较（含五层拼接比较）、7 个文本切片及 1 个直接 readText 切片、1 个拼接结果切片、1 个拼接切片长度、1 个切片拼接操作数、1 个拼接源切片操作数、4 个文本数组及 {len(argument_examples)} 个参数文本产物（{len(RUNTIME_TEXT_CASES) + 2 * len(RUNTIME_TEXT_LENGTH_CASES) + len(RUNTIME_TEXT_VARIABLE_CASES) + len(RUNTIME_TEXT_CONCAT_CASES) + len(RUNTIME_TEXT_CONCAT_OUTPUT_CASES) + len(RUNTIME_TEXT_CONCAT_LITERAL_CASES) + len(RUNTIME_TEXT_CONCAT_READ_CASES) + len(RUNTIME_TEXT_CONCAT_READ_SLICE_CASES) + len(RUNTIME_TEXT_ARRAY_SLICE_CASES) + len(RUNTIME_TEXT_ARRAY_SLICE_CONCAT_CASES) + len(RUNTIME_TEXT_COMPARISON_CASES) + len(RUNTIME_TEXT_STATIC_COMPARISON_CASES) + len(RUNTIME_TEXT_NOT_EQUAL_CASES) + len(RUNTIME_TEXT_SHORT_CIRCUIT_CASES) + len(RUNTIME_TEXT_ORDERING_CASES) + len(RUNTIME_TEXT_ORDER_EXPRESSION_CASES) + len(RUNTIME_TEXT_SLICE_CASES) + len(RUNTIME_TEXT_READ_SLICE_CASES) + len(RUNTIME_TEXT_ALIAS_SLICE_CASES) + len(RUNTIME_TEXT_DYNAMIC_SLICE_CASES) + len(RUNTIME_TEXT_CONCAT_SLICE_CASES) + len(RUNTIME_TEXT_CONCAT_SLICE_LENGTH_CASES) + len(RUNTIME_TEXT_PARAMETER_CASES) + len(OWNED_TEXT_RETURN_CASES) + len(OWNED_TEXT_RETURN_STORAGE_CASES) + 18} 个成功文本输入）、{56 + len(capability_cases) + len(loop_control_cases) + len(function_cases) + 2 * len(text_parameter_failures) + 2 * len(file_write_diagnostics) + 2 * len(boolean_type_failures) + 2 * len(map_type_failures) + 2 * len(result_type_failures) + 1} 个编译诊断及 {21 + len(DYNAMIC_FAILURES) + len(RESULT_FAILURE_CASES) + 3 * len(RUNTIME_TEXT_FAILURES) + len(RUNTIME_TEXT_VARIABLE_FAILURES) + len(RUNTIME_TEXT_CONCAT_FAILURES) + len(RUNTIME_TEXT_CONCAT_OUTPUT_FAILURES) + len(RUNTIME_TEXT_CONCAT_LITERAL_FAILURES) + len(RUNTIME_TEXT_CONCAT_READ_FAILURES) + len(RUNTIME_TEXT_CONCAT_READ_SLICE_FAILURES) + len(RUNTIME_TEXT_ARRAY_SLICE_FAILURES) + len(RUNTIME_TEXT_ARRAY_SLICE_CONCAT_FAILURES) + len(RUNTIME_TEXT_SLICE_FAILURES) + len(RUNTIME_TEXT_DYNAMIC_SLICE_FAILURES) + len(RUNTIME_TEXT_CONCAT_SLICE_FAILURES) + len(RUNTIME_TEXT_CONCAT_SLICE_FAILURES) + len(RUNTIME_TEXT_READ_SLICE_FAILURES) + len(file_write_failures) + dynamic_file_path_failure_count} 个运行时拒绝样例"
    )
    print("证明静态/运行时整数运算和六种比较、且/故短路逻辑、若/否分支、凡/跳出/继续嵌套控制流、有符号 64 位整数键值哈希映射的碰撞链、覆盖更新、3/4 负载几何扩容和 200 项重排、映射式动态整数和文本数组追加/增长/索引/更新及边界拒绝、最多 16 MiB 的动态 UTF-8 文本变量、输入与变量拼接/切片写回、最多五层的动态文本拼接和相等比较、63 字节根输出/切片、文本长度/相等/码位顺序比较、布尔值 1/0 输出及带标签的整数、编译期文本或直接运行时输入成功载荷、失败结果替换、复制、重赋值、错误码观察和有界失败传播、无效果块体运行时函数通过相邻指针/长度寄存器接收只读 UTF-8 文本参数并保存到调用帧槽位、以八寄存器展平预算调用并返回 textLength 整数或六种文本比较结果、由函数体文本证据或顶层调用实参确定比较参数 ABI、将文本参数或受支持拼接/切片结果复制到函数顶层文本局部并在统一收束点清理、将已初始化函数文本局部的所有权通过 x1/x2/x3 返回给程序根输出并由调用方释放、先在调用后保存返回描述符再释放旧目标映射并由顶层文本变量接管、递归纯整数函数独立保存帧指针/链接寄存器/整数参数槽并以寄存器传参和直接链接调用返回、自递归和互递归及 200 层调用、整数及编译期静态文本成功/失败结果的调用帧返回和失败码传播、命令行参数计数与静态或运行时整数下标 UTF-8 文本输出、参数文本存储和未存储拼接/切片/相等以及成功结果载荷，以及静态或参数文本动态路径文件直接输出、装入动态文本变量和从静态或参数文本动态路径覆盖写入后的有限原生输入，运行时文本结果的独立缓冲复制、拼接/切片/文本数组元素成功载荷及复制到普通文本变量或数组元素（含越界与负索引拒绝）；支持函数顶层文本参数本地副本；D-0151/D-0152/D-0153/D-0154 另外证明效果语句体函数可直接构造整数或文本结果局部（含拼接与切片文本载荷），以及单次引用的直接或显式效果调用可进入纯函数唯一形参位；仍不证明通用文本/容器结果载荷、文本返回在表达式/条件/函数体/容器中的嵌套存储或容器逃逸、完整效果函数运行时 ABI、通用调用求值顺序、跨函数恢复、容器通用载荷、通用键值类型、通用容器、通用文件/进程 IO 或完整 P1b")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
