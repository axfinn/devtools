#!/usr/bin/env python3
"""批量迁移 Tier 3 页面到 CSS 变量系统"""
import re

# 颜色替换映射 - 更全面的映射
COLOR_MAPPINGS = {
    # 背景色
    r'background-color:\s*#fff(fff)?;?': 'background-color: var(--bg-primary);',
    r'background-color:\s*#ffffff;?': 'background-color: var(--bg-primary);',
    r'background-color:\s*#f5f5f5;?': 'background-color: var(--bg-secondary);',
    r'background-color:\s*#fafafa;?': 'background-color: var(--bg-tertiary);',
    r'background-color:\s*#f0f0f0;?': 'background-color: var(--bg-secondary);',
    r'background-color:\s*#1e1e1e;?': 'background-color: var(--bg-primary);',
    r'background-color:\s*#2(5|d)2(5|d)2(5|d);?': 'background-color: var(--bg-secondary);',
    r'background-color:\s*#2a2a2a;?': 'background-color: var(--bg-tertiary);',
    r'background-color:\s*#1a1a1a;?': 'background-color: var(--bg-primary);',
    r'background-color:\s*#121212;?': 'background-color: var(--bg-base);',

    # 文字颜色
    r'color:\s*#333(333)?;?': 'color: var(--text-primary);',
    r'color:\s*#666(666)?;?': 'color: var(--text-secondary);',
    r'color:\s*#999(999)?;?': 'color: var(--text-tertiary);',
    r'color:\s*#e0e0e0;?': 'color: var(--text-primary);',
    r'color:\s*#d4d4d4;?': 'color: var(--text-primary);',
    r'color:\s*#a0a0a0;?': 'color: var(--text-secondary);',
    r'color:\s*#909399;?': 'color: var(--text-tertiary);',
    r'color:\s*#606266;?': 'color: var(--text-quaternary);',

    # 边框颜色
    r'border-color:\s*#e0e0e0;?': 'border-color: var(--border-base);',
    r'border-color:\s*#dcdfe6;?': 'border-color: var(--border-base);',
    r'border-color:\s*#d9d9d9;?': 'border-color: var(--border-base);',
    r'border-color:\s*#333(333)?;?': 'border-color: var(--border-base);',
    r'border-color:\s*#404040;?': 'border-color: var(--border-base);',
    r'border-color:\s*#444(444)?;?': 'border-color: var(--border-dark);',
    r'border:\s*1px solid #e0e0e0;?': 'border: 1px solid var(--border-base);',
    r'border:\s*1px solid #dcdfe6;?': 'border: 1px solid var(--border-base);',
    r'border:\s*1px solid #333;?': 'border: 1px solid var(--border-base);',
    r'border:\s*1px solid #d9d9d9;?': 'border: 1px solid var(--border-base);',

    # 特殊颜色
    r'color:\s*#4caf50;?': 'color: var(--color-success);',
    r'color:\s*#409eff;?': 'color: var(--color-primary);',
    r'color:\s*#f44336;?': 'color: var(--color-danger);',
    r'color:\s*#ff9800;?': 'color: var(--color-warning);',
    r'color:\s*#67c23a;?': 'color: var(--color-success);',

    # 字体
    r"font-family:\s*['\"]Consolas['\"],\s*['\"]Monaco['\"],\s*monospace;?": "font-family: var(--font-family-mono);",
    r"font-family:\s*['\"]Consolas['\"],\s*['\"]Monaco['\"],\s*['\"]Courier New['\"],\s*monospace;?": "font-family: var(--font-family-mono);",
    r"font-family:\s*['\"]Courier New['\"],\s*Consolas,\s*Monaco,\s*monospace;?": "font-family: var(--font-family-mono);",

    # 圆角
    r'border-radius:\s*8px;?': 'border-radius: var(--radius-md);',
    r'border-radius:\s*6px;?': 'border-radius: var(--radius-sm);',
    r'border-radius:\s*4px;?': 'border-radius: var(--radius-sm);',
    r'border-radius:\s*12px;?': 'border-radius: var(--radius-lg);',
}

def remove_dark_rules(content):
    """删除所有 :global(.dark) 和 html.dark 规则块"""
    lines = content.split('\n')
    result = []
    in_dark_block = False
    brace_count = 0

    for line in lines:
        # 检测是否是 dark 规则的开始
        if re.search(r'(:global\(\.dark\)|html\.dark)\s+.*\{', line):
            in_dark_block = True
            brace_count = line.count('{') - line.count('}')
            continue

        if in_dark_block:
            brace_count += line.count('{') - line.count('}')
            if brace_count <= 0:
                in_dark_block = False
            continue

        result.append(line)

    return '\n'.join(result)

def replace_colors(content):
    """替换颜色为 CSS 变量"""
    for pattern, replacement in COLOR_MAPPINGS.items():
        content = re.sub(pattern, replacement, content)
    return content

def migrate_file(filepath):
    """迁移单个文件"""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        # 先删除 dark 规则
        content = remove_dark_rules(content)

        # 然后替换颜色
        content = replace_colors(content)

        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)

        print(f"✓ 已迁移 {filepath}")
        return True
    except Exception as e:
        print(f"✗ 迁移 {filepath} 失败: {e}")
        return False

# 迁移文件列表
files = [
    'src/views/ChatRoom.vue',
    'src/views/ExcalidrawTool.vue',
    'src/views/ExcalidrawShareView.vue',
    'src/views/MarkdownShareView.vue',
    'src/views/ShortUrl.vue',
]

if __name__ == '__main__':
    import os
    os.chdir('/home/hejiahao01/code/bili/devtools/frontend')

    success_count = 0
    for filepath in files:
        if migrate_file(filepath):
            success_count += 1

    print(f"\n完成！成功迁移 {success_count}/{len(files)} 个文件")
