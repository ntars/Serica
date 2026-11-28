#!/usr/bin/env python3
"""Serica 首检脚本（M0-04 交付物）

检查顺序与退出条件：
1. 半角符号（0-9、数字内 . 除外）→ 阻断
2. 空格 / Tab → 阻断
3. 行首空白（缩进）→ 阻断
4. 行尾空白 → 阻断
5. 词表外「XX：」标签 → 阻断
6. 表达式括号（非整行注释）→ 警告
7. 容器段数量 vs 段结束。数量 → 阻断
8. 输出：段数量 ≠ 1 → 阻断
9. 文件末尾无换行 → 阻断
10. 汇编段外的 汇编结束。→ 阻断

退出码：0=通过，1=失败
"""

import sys
import re

# 词表内所有带「：」的段标签（28个）
VALID_LABELS = {
    '导入', '常数', '类型', '记录', '枚举', '联合', '输入', '设定',
    '导出', '锁', '线程', '通道', '函数', '通用', '规则', '循环',
    '匹配', '情形', '其他', '尝试', '出错', '收尾', '并行', '系统',
    '汇编', '例', '输出'
}

def check_halfwidth_symbols(line, line_num):
    """检查半角符号（0-9 和数字内的小数点 . 除外）"""
    # 允许：0-9, 数字内的小数点
    # 禁止：半角逗号、冒号、分号、括号、方括号、花括号、引号、感叹号、问号、等号、加号、减号、乘号、除号、小于、大于、和、或、井号、at、美元、百分号、caret、波浪号、反斜杠
    forbidden = r'[,:;()\[\]{}"\'!?=+\-*/<>&|#@$%^~\\]'
    # 但允许数字内的小数点（如 3.14）
    # 简单处理：如果半角符号不在数字上下文中，则报错
    matches = re.finditer(forbidden, line)
    for m in matches:
        char = m.group()
        # 检查是否是数字内的小数点
        if char == '.':
            # 看前后是否有数字
            start = max(0, m.start() - 1)
            end = min(len(line), m.end() + 1)
            context = line[start:end]
            if re.match(r'\d+\.\d+', context) or re.match(r'\d+\.', context) or re.match(r'\.\d+', context):
                continue  # 数字内的小数点，允许
        col = line[:m.start()].count('') + 1  # 简单列号计算
        print(f"L{line_num} 第{col}列 半角「{char}」")
        return True
    return False

def check_spaces(line, line_num):
    """检查空格或 Tab"""
    if re.search(r'[ \t]', line):
        col = line[:line.index(' ' if ' ' in line else '\t')].count('') + 1
        print(f"L{line_num} 第{col}列 空格")
        return True
    return False

def check_leading_whitespace(line, line_num):
    """检查行首空白（缩进）"""
    if line and (line[0] == ' ' or line[0] == '\t'):
        print(f"L{line_num} 行首缩进")
        return True
    return False

def check_trailing_whitespace(line, line_num):
    """检查行尾空白"""
    if line.rstrip() != line:
        col = len(line) - len(line.rstrip()) + 1
        print(f"L{line_num} 行尾空白")
        return True
    return False

def check_unknown_labels(line, line_num):
    """检查词表外「XX：」标签"""
    m = re.match(r'^([^\s〈】》，。；：]+)：', line)
    if m:
        label = m.group(1).strip()
        if label not in VALID_LABELS and not line.startswith('《') and '：' not in label:
            print(f"L{line_num} 未知段标签「{label}：」")
            return True
    return False

def check_expression_parentheses(line, line_num):
    """检查表达式括号（非整行注释）"""
    # 如果整行是注释，跳过
    if line.strip().startswith('（') and line.strip().endswith('）'):
        return False
    # 检查是否有非注释的括号
    if re.search(r'[（][^）]*[）]', line) and not (line.strip().startswith('（') and line.strip().endswith('）')):
        print(f"L{line_num} 疑似表达式括号")
        return True
    return False

def check_container_vs_ending(lines):
    """检查容器段数量 vs 段结束。数量"""
    container_count = 0
    ending_count = 0
    for line in lines:
        stripped = line.strip()
        if re.match(r'^(导入|常数|类型|记录|枚举|联合|输入|设定|导出|锁|线程|通道|函数|通用|规则|循环|匹配|尝试|并行|系统)：', stripped):
            container_count += 1
        if stripped == '段结束。':
            ending_count += 1
    if container_count != ending_count:
        print(f"容器段 {container_count} 个，段结束 {ending_count} 个")
        return True
    return False

def check_output_sections(lines):
    """检查输出：段数量 ≠ 1"""
    output_count = sum(1 for line in lines if line.strip().startswith('输出：'))
    if output_count != 1:
        print(f"输出段 {output_count} 个")
        return True
    return False

def check_assembly_ending(lines):
    """检查汇编段外的 汇编结束。"""
    in_assembly = False
    for i, line in enumerate(lines):
        stripped = line.strip()
        if stripped.startswith('汇编：'):
            in_assembly = True
        elif stripped == '段结束。' and in_assembly:
            in_assembly = False
        elif stripped == '汇编结束。' and not in_assembly:
            print(f"L{i+1} 孤立「汇编结束。」")
            return True
    return False

def main():
    if len(sys.argv) < 2:
        print("用法：python 首检.py <文件>")
        sys.exit(1)

    filename = sys.argv[1]
    try:
        with open(filename, 'r', encoding='utf-8') as f:
            lines = f.readlines()
    except FileNotFoundError:
        print(f"文件未找到：{filename}")
        sys.exit(1)

    has_error = False

    # 检查 9：文件末尾无换行
    if lines and not lines[-1].endswith('\n'):
        print("文件末尾缺换行")
        has_error = True

    for i, line in enumerate(lines):
        line_num = i + 1
        stripped = line.rstrip('\n')

        # 检查 1：半角符号
        if check_halfwidth_symbols(stripped, line_num):
            has_error = True

        # 检查 2：空格 / Tab
        if check_spaces(stripped, line_num):
            has_error = True

        # 检查 3：行首空白（缩进）
        if check_leading_whitespace(stripped, line_num):
            has_error = True

        # 检查 4：行尾空白
        if check_trailing_whitespace(stripped, line_num):
            has_error = True

        # 检查 5：词表外标签
        if check_unknown_labels(stripped, line_num):
            has_error = True

        # 检查 6：表达式括号（警告）
        if check_expression_parentheses(stripped, line_num):
            pass  # 只警告，不阻断

    # 检查 7：容器段 vs 段结束
    if check_container_vs_ending(lines):
        has_error = True

    # 检查 8：输出段数量
    if check_output_sections(lines):
        has_error = True

    # 检查 10：汇编结束。
    if check_assembly_ending(lines):
        has_error = True

    sys.exit(1 if has_error else 0)

if __name__ == '__main__':
    main()
