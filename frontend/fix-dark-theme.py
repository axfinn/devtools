#!/usr/bin/env python3
import re

# 读取 theme.css
with open('src/theme.css', 'r', encoding='utf-8') as f:
    content = f.read()

# 处理 .dark 规则块，为所有属性添加 !important（如果还没有的话）
def add_important_to_dark_rules(text):
    lines = text.split('\n')
    result = []
    in_dark_rule = False

    for line in lines:
        # 检测是否进入 .dark 规则块
        if re.search(r'\.dark\s+', line) and '{' in line:
            in_dark_rule = True
            result.append(line)
        elif in_dark_rule:
            # 检测是否离开规则块
            if line.strip() == '}':
                in_dark_rule = False
                result.append(line)
            # 处理属性行
            elif ':' in line and ';' in line:
                # 如果已经有 !important，跳过
                if '!important' in line:
                    result.append(line)
                else:
                    # 在 ; 前添加 !important
                    modified_line = line.replace(';', ' !important;')
                    result.append(modified_line)
            else:
                result.append(line)
        else:
            result.append(line)

    return '\n'.join(result)

# 应用修改
new_content = add_important_to_dark_rules(content)

# 写回文件
with open('src/theme.css', 'w', encoding='utf-8') as f:
    f.write(new_content)

print("✓ 已为所有深色模式规则添加 !important")
