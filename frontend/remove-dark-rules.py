#!/usr/bin/env python3
"""删除 CSS 文件中所有 .dark 相关的规则"""
import re

def remove_dark_rules(filename):
    with open(filename, 'r', encoding='utf-8') as f:
        content = f.read()

    # 删除所有 .dark 规则块（包括多行）
    # 匹配模式：.dark 开头的选择器，直到对应的 }
    lines = content.split('\n')
    result = []
    in_dark_block = False
    brace_count = 0

    for line in lines:
        # 检测是否是 .dark 规则的开始
        if re.search(r'\.dark\s+.*\{', line) or re.search(r'^\.dark\s*\{', line):
            in_dark_block = True
            brace_count = line.count('{') - line.count('}')
            continue

        if in_dark_block:
            brace_count += line.count('{') - line.count('}')
            if brace_count <= 0:
                in_dark_block = False
            continue

        result.append(line)

    # 写回文件
    with open(filename, 'w', encoding='utf-8') as f:
        f.write('\n'.join(result))

    print(f"✓ 已删除 {filename} 中所有 .dark 规则")

# 处理两个文件
remove_dark_rules('src/style.css')
remove_dark_rules('src/theme.css')
