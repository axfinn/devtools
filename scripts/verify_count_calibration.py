#!/usr/bin/env python3
"""FINN-222 · Stage 5 独立复验：AGENTS.md / CLAUDE.md 计数校准（FINN-217 / FINN-220）。

复验三件事（对每条计数断言）：
  T1  现算：判据命令在 REF（默认 origin/main）上跑出的值 == 改后文档写的值
  T2  反事实：改前文档写的值 != 现算值（⇒ 该断言有区分力，不是恒真闸门）
  T2' 注入：把改后文档里的数字故意改错 ⇒ 闸门必须判红；还原 ⇒ 必须判绿

只读脚本：只跑 git / grep，不改仓库内容（注入实验在内存副本上做）。
用法：
  python3 scripts/verify_count_calibration.py --repo /path/to/clone \
      --ref origin/main --base <改前 commit> --tip <改后 commit>
"""

import argparse
import os
import re
import subprocess
import sys

# ---------------------------------------------------------------- 口径命令块
# 全部来自 CLAUDE.md「计数同步清单」；${REF} 由运行时注入。
CRIT = {
    "C1": "git show ${REF}:AGENTS.md | grep -cE '^\\| *[0-9]+ *\\|'",
    "C2": "git ls-tree -r --name-only ${REF} -- frontend/src/views/ | grep -c '\\.vue$'",
    "C3": "git ls-tree -r --name-only ${REF} -- frontend/src/ | grep '\\.vue$' | grep -vc '/views/'",
    "C5": "git ls-tree -r --name-only ${REF} -- backend/routes/ | grep -c '\\.go$'",
    "C6": ("git ls-tree --name-only ${REF}:backend/routes/ | grep '\\.go$' | "
           "grep -v '_test\\.go$' | grep -vc '^index\\.go$'"),
    "C7": "git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep -c '\\.go$'",
    "C7t": "git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep '\\.go$' | grep -c '_test\\.go$'",
    "C9": "git show ${REF}:backend/handlers/skills.go | grep -cE '^[[:space:]]*Name:[[:space:]]+\"'",
    "AI_GW": (
        "A=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\\.go$'); "
        "B=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep '^minimax' | grep -vc '_test\\.go$'); "
        "C=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep -cE '^(image_understanding|planner_ai)\\.go$'); "
        "echo $((A+B+C))"),
    "C11": ("echo $(( $(git ls-tree -r --name-only ${REF} -- frontend/src/views/dev/VirtualAvatarTool "
            "| grep -c '\\.vue$') - 1 ))"),
    "COMPOSE": ("git show ${REF}:docker-compose.yml | awk '/^services:/{f=1;next} /^volumes:/{f=0} "
                "f && /^  [a-zA-Z0-9_-]+:/{c++} END{print c}'"),
}


def crit(prefix):
    """按前缀返回口径命令：'C8-nfsshare' / 'C4-ai' / 'C10-household'。"""
    if prefix in CRIT:
        return CRIT[prefix]
    if prefix.startswith("C8-"):
        p = prefix[3:]
        return ("git ls-tree --name-only ${REF}:backend/handlers/ | grep '^%s' | grep -vc '_test\\.go$'" % p)
    if prefix.startswith("C4-"):
        cat = prefix[3:]
        return ("git ls-tree -r --name-only ${REF} -- frontend/src/views/%s | grep -c '\\.vue$'" % cat)
    if prefix.startswith("C10-"):
        f = prefix[4:]
        return ("echo $(( $(git show ${REF}:backend/handlers/%s.go | grep -oE '[a-z_]+/section[0-9]+' "
                "| sort -u | wc -l) + 1 ))" % f)
    raise KeyError(prefix)


# ------------------------------------------------------- 断言清单
# id / file / line_re / group / crit / exempt(豁免「改前值 != 现算值」) / note
CHANGED = [
    # ---- AGENTS.md ----
    ("A1", "AGENTS.md", r"^## (\d+) 模块地图$", 1, "C1", False, "标题模块地图数"),
    ("A2", "AGENTS.md", r"^- \*\*前端\*\*：.*（(\d+) 个 View 组件", 1, "C2", False, "技术栈 View 组件数"),
    ("A2b", "AGENTS.md", r"^- \*\*前端\*\*：.*\+ (\d+) 个公共组件）", 1, "C3", True,
     "公共组件数（P2 豁免：前后值不变）"),
    ("A3", "AGENTS.md", r"^### 前端（(\d+) View 组件）$", 1, "C2", False, "项目结构-前端 View 数"),
    ("A4", "AGENTS.md", r"^├── ai/         # AI 相关（(\d+)）", 1, "C4-ai", False, "category ai"),
    ("A5", "AGENTS.md", r"^├── collab/     # 协作（(\d+)）", 1, "C4-collab", False, "category collab"),
    ("A6", "AGENTS.md", r"^├── convert/    # 转换（(\d+)）", 1, "C4-convert", False, "category convert"),
    ("A7", "AGENTS.md", r"^├── dev/        # 开发（(\d+)）", 1, "C4-dev", False, "category dev"),
    ("A9", "AGENTS.md", r"^├── household/  # 家庭（(\d+) 个子组件）", 1, "C4-household", False, "category household"),
    ("A10", "AGENTS.md", r"^├── life/       # 生活（(\d+)）", 1, "C4-life", False, "category life"),
    ("A11", "AGENTS.md", r"^├── other/      # 其他（(\d+)）", 1, "C4-other", False, "category other"),
    ("A12", "AGENTS.md", r"^└── share/      # 分享视图（(\d+)）", 1, "C4-share", False, "category share"),
    ("A13", "AGENTS.md", r"^### 后端路由拆分（routes/ (\d+) 个文件）$", 1, "C5", False, "routes 文件数（标题）"),
    ("A14", "AGENTS.md", r"^├── AI 网关（(\d+)）：", 1, "AI_GW", False, "AI 网关分组总数"),
    ("A15", "AGENTS.md", r"^├── AI 网关（\d+）：ai_gateway\*（(\d+)，", 1, "C8-ai_gateway", False, "ai_gateway* 文件数"),
    ("A18", "AGENTS.md", r"^\| 23 \| \*\*NFSShare\*\*.*nfsshare\*\.go` \((\d+) 文件\)", 1, "C8-nfsshare", False,
     "模块表 NFSShare 文件数"),
    ("A0", "AGENTS.md", r"^\| 25 \| \*\*AIGateway\*\*.*ai_gateway\*\.go` \((\d+) 文件", 1, "C8-ai_gateway", False,
     "模块表 AIGateway 文件数（父单 #7；方案 §3.1 表 A 漏列，实施方已改）"),
    ("A20", "AGENTS.md", r"^\| 37 \| \*\*Skills\+MCP\*\*.*skills\*\.go` \((\d+) 文件\)", 1, "C8-skills", False,
     "模块表 Skills 文件数"),
    ("A21", "AGENTS.md", r"^\| 37 \| \*\*Skills\+MCP\*\*.*，(\d+) 个 skill", 1, "C9", False, "内置 skill 数"),
    ("A22a", "AGENTS.md", r"^- \*\*路由拆分\*\*：`routes\.go` → `routes/` 包（(\d+) 文件，", 1, "C5", False,
     "路由拆分-文件数"),
    ("A22b", "AGENTS.md", r"^- \*\*路由拆分\*\*：`routes\.go` → `routes/` 包（\d+ 文件，(\d+) 分组）", 1, "C6", False,
     "路由拆分-分组数"),
    ("A23", "AGENTS.md", r"^6\. 更新本文件 `(\d+) 模块地图` 表格$", 1, "C1", False, "流程第 6 步模块数"),
    ("A24", "AGENTS.md", r"^└── 测试（(\d+)\+?）：\*_test\.go$", 1, "C7t", False, "handlers 测试文件数"),
    # ---- CLAUDE.md ----
    ("B1", "CLAUDE.md", r"^DevTools 项目快速参考。.*`AGENTS\.md`（(\d+) 模块地图、", 1, "C1", False,
     "CLAUDE 头部模块地图数"),
    ("B2", "CLAUDE.md", r"^- \*\*前端\*\*：.*（(\d+) 个 View 组件", 1, "C2", False, "CLAUDE 技术栈 View 组件数"),
    ("B2b", "CLAUDE.md", r"^- \*\*前端\*\*：.*\+ (\d+) 个公共组件）", 1, "C3", True,
     "公共组件数（P2 豁免：前后值不变）"),
    ("B3", "CLAUDE.md", r"^frontend/src/views/       # (\d+) Vue 组件", 1, "C2", False, "项目结构速查 View 数"),
    ("B4", "CLAUDE.md", r"^├── routes/              # (\d+) 个路由文件", 1, "C5", False, "routes 文件数（速查）"),
    ("B5", "CLAUDE.md", r"^├── handlers/            # (\d+) 个 handler 文件", 1, "C7", False, "handlers 文件数"),
    ("B6a", "CLAUDE.md", r"^\| \*\*路由拆分\*\* \| `routes\.go` → `routes/` 包（(\d+) 文件，", 1, "C5", False,
     "关键设计模式-路由拆分文件数"),
    ("B6b", "CLAUDE.md", r"^\| \*\*路由拆分\*\* \| `routes\.go` → `routes/` 包（\d+ 文件，(\d+) 分组）", 1, "C6", False,
     "关键设计模式-路由拆分组数"),
    ("B7", "CLAUDE.md", r"^6\. 更新 `AGENTS\.md` (\d+) 模块地图$", 1, "C1", False, "流程第 6 步模块数"),
]

# 本单未改、但方案 §3.1 标「不改」的断言：只验「当前成立」（T1），不参与反事实。
UNCHANGED = [
    ("U1", "AGENTS.md", r"^├── draw/       # 绘图（(\d+)）", 1, "C4-draw", "category draw"),
    ("U2", "AGENTS.md", r"^├── AI 网关（\d+）：.*minimax_\*（(\d+)）", 1, "C8-minimax", "minimax_* 文件数"),
    ("U3", "AGENTS.md", r"^\| 20 \| \*\*Planner 待办\*\*.*planner\*\.go` \((\d+) 文件\)", 1, "C8-planner",
     "模块表 Planner 文件数"),
    ("U4", "AGENTS.md", r"^\| 26 \| \*\*Askit 同步\*\*.*askit\*\.go` \((\d+) 文件\)", 1, "C8-askit",
     "模块表 Askit 文件数"),
    ("U5", "AGENTS.md", r"^\| 22 \| \*\*Household 家庭\*\*.*household\.go` \((\d+) 分段\)", 1, "C10-household",
     "模块表 Household 分段数"),
    ("U6", "AGENTS.md", r"^\| 29 \| \*\*AutoDev\*\*.*autodev\.go` \((\d+) 分段\)", 1, "C10-autodev",
     "模块表 AutoDev 分段数"),
    ("U7", "AGENTS.md", r"^\| 31 \| \*\*Proxy 代理\*\*.*proxy\.go` \((\d+) 分段\)", 1, "C10-proxy",
     "模块表 Proxy 分段数"),
    ("U8", "AGENTS.md", r"^\| 40 \| \*\*Avatar 虚拟形象\*\*.*\(\+ (\d+) 子组件\)", 1, "C11", "Avatar 子组件数"),
    ("U9", "AGENTS.md", r"^- \*\*部署\*\*：Docker Compose（(\d+) 服务", 1, "COMPOSE", "compose 服务数"),
    ("U10", "AGENTS.md", r"^- \*\*前端\*\*：.*\+ (\d+) 个公共组件）", 1, "C3", "公共组件数（AGENTS.md）"),
    ("U11", "CLAUDE.md", r"^- \*\*前端\*\*：.*\+ (\d+) 个公共组件）", 1, "C3", "公共组件数（CLAUDE.md）"),
]


class Runner(object):
    def __init__(self, repo, ref):
        self.repo = repo
        self.env = dict(os.environ)
        self.env["REF"] = ref

    def sh(self, cmd):
        p = subprocess.run(["bash", "-c", cmd], cwd=self.repo, env=self.env,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        return p.returncode, p.stdout.decode("utf-8", "replace").strip(), \
            p.stderr.decode("utf-8", "replace").strip()

    def show(self, rev, path):
        rc, out, err = self.sh("git show %s:%s" % (rev, path))
        if rc != 0:
            raise RuntimeError("git show %s:%s failed: %s" % (rev, path, err))
        return out


def find_one(content, pattern, group):
    """在 content 里找 line_re 唯一匹配的行，返回该捕获组的数字；找不到/多个匹配都报错。"""
    hits = [m for m in re.finditer(pattern, content, re.M) if m.group(group) is not None]
    if len(hits) != 1:
        raise RuntimeError("line_re %r 匹配 %d 行（要求恰好 1 行）" % (pattern, len(hits)))
    return int(hits[0].group(group)), hits[0]


def inject(content, pattern, group, wrong):
    """把唯一匹配行的第 group 个捕获组替换成 wrong，返回新内容。"""
    hits = [m for m in re.finditer(pattern, content, re.M) if m.group(group) is not None]
    if len(hits) != 1:
        raise RuntimeError("注入失败：line_re %r 匹配 %d 行" % (pattern, len(hits)))
    m = hits[0]
    s, e = m.span(group)
    return content[:s] + str(wrong) + content[e:]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--repo", required=True, help="含 origin/main + 上游 refs 的 git 仓库（建议 /tmp 完整克隆）")
    ap.add_argument("--ref", default="origin/main", help="现算基准 ref（默认 origin/main）")
    ap.add_argument("--base", required=True, help="改前 commit（等价于改前 main）")
    ap.add_argument("--tip", required=True, help="改后 commit（实施方产品 commit 所在 ref）")
    args = ap.parse_args()

    r = Runner(args.repo, args.ref)
    fails = []
    rows = []

    def check(cond, label, detail):
        rows.append((("PASS" if cond else "FAIL"), label, detail))
        if not cond:
            fails.append(label)

    # ---------- 0. 前置控制：BASE 的两个文档 == origin/main 的两个文档 ----------
    print("=" * 100)
    print("§0 前置控制：BASE 是否等价于「改前 main」")
    print("=" * 100)
    for path in ("AGENTS.md", "CLAUDE.md"):
        rc, out, _ = r.sh("git rev-parse %s:%s" % (args.base, path))
        rc2, out2, _ = r.sh("git rev-parse %s:%s" % (args.ref, path))
        same = (out == out2)
        check(same, "base-blob-equal %s" % path,
              "%s:%s blob=%s | %s:%s blob=%s => %s" % (args.base[:7], path, out[:12],
                                                      args.ref, path, out2[:12],
                                                      "一致（BASE 可当改前 main）" if same else "不一致"))
        print("%-8s %-24s %s" % ("PASS" if same else "FAIL", "base-blob-equal " + path,
                                 out[:12] + " vs " + out2[:12]))

    docs_base = {p: r.show(args.base, p) for p in ("AGENTS.md", "CLAUDE.md")}
    docs_tip = {p: r.show(args.tip, p) for p in ("AGENTS.md", "CLAUDE.md")}

    # ---------- 1. T1 + T2 + 注入实验：改值断言全集 ----------
    print()
    print("=" * 100)
    print("§1 改值断言全集：现算(T1) / 反事实(T2) / 注入实验(T2')")
    print("=" * 100)
    print("%-6s %-6s %-9s %-9s %-9s %-8s %-8s %s" %
          ("ID", "文件", "现算值", "改前值", "改后值", "T1", "T2反事实", "注入实验"))
    for (aid, path, lre, grp, cid, exempt, note) in CHANGED:
        rc, computed, err = r.sh(crit(cid))
        if rc != 0:
            check(False, "%s criterion" % aid, "命令失败 rc=%d %s" % (rc, err))
            continue
        computed = int(computed)
        pre, _ = find_one(docs_base[path], lre, grp)
        post, _ = find_one(docs_tip[path], lre, grp)

        t1 = (post == computed)
        t2 = True if exempt else (pre != computed)

        # 注入实验：把改后文档的数字改错 ⇒ 闸门必须判红；还原 ⇒ 必须判绿
        wrong = post + 1 if post != computed else computed + 1
        inj = inject(docs_tip[path], lre, grp, wrong)
        inj_val, _ = find_one(inj, lre, grp)
        inject_red = (inj_val != computed)          # 注入后闸门判红
        restore_green = (post == computed)          # 还原后闸门判绿
        inj_ok = inject_red and restore_green and (inj_val == wrong)

        ok = t1 and t2 and inj_ok
        check(ok, "%s %s" % (aid, note),
              "computed=%d pre=%d post=%d t1=%s t2=%s inject(%d)->%s restore=%s" %
              (computed, pre, post, t1, t2, wrong,
               "RED" if inject_red else "GREEN", "GREEN" if restore_green else "RED"))
        print("%-6s %-6s %-9d %-9d %-9d %-8s %-8s %s | %s" %
              (aid, path, computed, pre, post,
               "PASS" if t1 else "FAIL",
               ("PASS" if t2 else "FAIL") + ("(豁免)" if exempt else ""),
               ("PASS" if inj_ok else "FAIL") + " 注入=%d→%s" % (wrong, "RED" if inject_red else "GREEN"),
               note))

    # ---------- 2. 未改动断言：只验当前成立 ----------
    print()
    print("=" * 100)
    print("§2 未改动的断言（方案 §3.1 标「不改」）：只验现算 == 文档值")
    print("=" * 100)
    for (aid, path, lre, grp, cid, note) in UNCHANGED:
        rc, computed, err = r.sh(crit(cid))
        if rc != 0:
            check(False, "%s criterion" % aid, "命令失败 rc=%d %s" % (rc, err))
            continue
        computed = int(computed)
        pre, _ = find_one(docs_base[path], lre, grp)
        post, _ = find_one(docs_tip[path], lre, grp)
        t1 = (post == computed)
        wrong = post + 1
        inj = inject(docs_tip[path], lre, grp, wrong)
        inj_val, _ = find_one(inj, lre, grp)
        inj_ok = (inj_val != computed) and (post == computed)
        check(t1 and inj_ok, "%s %s" % (aid, note),
              "computed=%d pre=%d post=%d t1=%s inject=%d->%s" %
              (computed, pre, post, t1, wrong, "RED" if inj_val != computed else "GREEN"))
        print("%-6s %-6s %-9d %-9d %-9d %-8s %-20s %s" %
              (aid, path, computed, pre, post, "PASS" if t1 else "FAIL",
               "注入=%d→%s" % (wrong, "RED" if inj_val != computed else "GREEN"), note))

    # ---------- 3. T3 越权检查 ----------
    print()
    print("=" * 100)
    print("§3 T3 越权检查（实施方产品 commit 的改动面）")
    print("=" * 100)
    rc, out, _ = r.sh("git show --name-only --format= %s" % args.tip)
    changed_files = [l for l in out.splitlines() if l.strip()]
    only_two = sorted(changed_files) == ["AGENTS.md", "CLAUDE.md"]
    check(only_two, "T3 overreach", "product commit 改动文件 = %s" % changed_files)
    print("%-8s product commit 改动文件 = %s (期望恰好 AGENTS.md + CLAUDE.md)" %
          ("PASS" if only_two else "FAIL", changed_files))

    rc, out, err = r.sh("git diff --numstat %s..%s" % (args.base, args.tip))
    check(rc == 0, "T3 numstat", "rc=%d %s" % (rc, err))
    print("        diff --numstat base..tip:\n" + "\n".join("          " + l for l in out.splitlines()))

    rc, out, _ = r.sh("git log --oneline %s..%s | grep -v 'chore(agent): baseline' | wc -l" % (args.ref, args.tip))
    n = out.strip()
    check(n == "1", "T3 commit-count", "非 baseline commit 数 = %s（期望 1）" % n)
    print("%-8s 非 baseline commit 数 = %s（期望 1）" % ("PASS" if n == "1" else "FAIL", n))

    rc, out, _ = r.sh("git show %s:CLAUDE.md | grep -c MULTICA-RUNTIME" % args.tip)
    mr = out.strip()
    check(mr == "0", "T3 no-runtime-block", "CLAUDE.md MULTICA-RUNTIME 计数 = %s（期望 0）" % mr)
    print("%-8s CLAUDE.md MULTICA-RUNTIME 计数 = %s（期望 0）" % ("PASS" if mr == "0" else "FAIL", mr))

    # ---------- 4. T4 同步清单可执行性（逐字复制 CLAUDE.md 里的 bash 块） ----------
    print()
    print("=" * 100)
    print("§4 T4 同步清单可执行性：把 CLAUDE.md 新增的 bash 块逐字抽出来跑")
    print("=" * 100)
    blocks = re.findall(r"```bash\n(.*?)```", docs_tip["CLAUDE.md"], re.S)
    checklist_blk = [b for b in blocks if "ai_gateway 非测试" in b]
    mustrun_blk = [b for b in blocks if "两个数必须相等" in b]
    check(len(checklist_blk) == 1, "T4 checklist-block-found",
          "CLAUDE.md 里含「ai_gateway 非测试」的 bash 块数 = %d（期望 1）" % len(checklist_blk))
    check(len(mustrun_blk) == 1, "T4 mustrun-block-found",
          "CLAUDE.md 里含「两个数必须相等」的 bash 块数 = %d（期望 1）" % len(mustrun_blk))
    print("%-8s 计数同步清单 bash 块数 = %d（期望 1）" %
          ("PASS" if len(checklist_blk) == 1 else "FAIL", len(checklist_blk)))
    print("%-8s 「每次改完必跑」bash 块数 = %d（期望 1）" %
          ("PASS" if len(mustrun_blk) == 1 else "FAIL", len(mustrun_blk)))

    # 计数同步清单：bash + zsh 两种 shell 各跑一遍（zsh 验 ${REF}:path 花括号没被 :A 修饰符吃掉）
    for shell in ("bash", "zsh"):
        if checklist_blk:
            blk = checklist_blk[0]
            p = subprocess.run([shell, "-c", blk], cwd=r.repo,
                               env=r.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            out = p.stdout.decode("utf-8", "replace").strip()
            err = p.stderr.decode("utf-8", "replace").strip()
            ok = (p.returncode == 0 and out != "" and "not a valid object name" not in err)
            check(ok, "T4 checklist-block %s" % shell,
                  "rc=%d 输出 %d 行" % (p.returncode, len(out.splitlines())))
            print("%-8s 计数同步清单（%s）: rc=%d，输出 %d 行" %
                  ("PASS" if ok else "FAIL", shell, p.returncode, len(out.splitlines())))
            for line in out.splitlines():
                print("          " + line)
            if err:
                print("          stderr: " + err)

    # 「每次改完必跑」块：它读**工作区** AGENTS.md ⇒ 先把改后文档铺到工作树再跑（模拟真实场景）
    rc, _, err = r.sh("git checkout %s -- AGENTS.md CLAUDE.md" % args.tip)
    if rc != 0:
        check(False, "T4 materialize-worktree", "git checkout 失败: %s" % err)
    else:
        for shell in ("bash", "zsh"):
            if not mustrun_blk:
                continue
            blk = mustrun_blk[0]
            p = subprocess.run([shell, "-c", blk], cwd=r.repo,
                               env=r.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            out = p.stdout.decode("utf-8", "replace").strip()
            err = p.stderr.decode("utf-8", "replace").strip()
            nums = re.findall(r"^\d+$", out, re.M)
            # 两个数必须相等：工作区标题数字 == ${REF}:AGENTS.md 模块表行数
            rc2, title, _ = r.sh("grep -E '^## [0-9]+ 模块地图' AGENTS.md")
            title_num = re.search(r"## (\d+) 模块地图", title)
            t = int(title_num.group(1)) if title_num else None
            tbl = int(nums[-1]) if nums else None
            ok = (p.returncode == 0 and t is not None and tbl is not None and t == tbl
                  and "No such file" not in err and "not a valid object name" not in err)
            check(ok, "T4 mustrun-block %s" % shell,
                  "rc=%d 工作区标题=%s 表行数=%s %s" % (p.returncode, t, tbl, err))
            print("%-8s 「每次改完必跑」（%s）: rc=%d，标题数字=%s，模块表行数=%s" %
                  ("PASS" if ok else "FAIL", shell, p.returncode, t, tbl))
            for line in out.splitlines():
                print("          " + line)
            if err:
                print("          stderr: " + err)

    # REF=HEAD 变体：清单声明「改自己的分支时用 HEAD」
    if checklist_blk:
        env2 = dict(r.env)
        env2["REF"] = "HEAD"
        p = subprocess.run(["bash", "-c", checklist_blk[0]], cwd=r.repo, env=env2,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        out = p.stdout.decode("utf-8", "replace").strip()
        ok = (p.returncode == 0 and out != "")
        check(ok, "T4 checklist-block REF=HEAD", "rc=%d 输出 %d 行" % (p.returncode, len(out.splitlines())))
        print("%-8s 计数同步清单（REF=HEAD）: rc=%d，输出 %d 行" %
              ("PASS" if ok else "FAIL", p.returncode, len(out.splitlines())))
        for line in out.splitlines():
            print("          " + line)

    # ---------- 5. T5 内部一致性 ----------
    print()
    print("=" * 100)
    print("§5 T5 内部一致性")
    print("=" * 100)
    groups = [
        ("模块地图数（5 处全等）", ["A1", "A23", "B1", "B7"]),
        ("routes 文件数（4 处全等）", ["A13", "A22a", "B4", "B6a"]),
        ("View 总数（4 处全等）", ["A2", "A3", "B2", "B3"]),
        ("公共组件数（2 处全等）", ["A2b", "B2b"]),
        ("AI 网关 ai_gateway 文件数（2 处全等）", ["A0", "A15"]),
    ]
    vals = {}
    for (aid, path, lre, grp, cid, exempt, note) in CHANGED:
        post, _ = find_one(docs_tip[path], lre, grp)
        vals[aid] = post
    for (name, ids) in groups:
        ids = [i for i in ids if i in vals]
        got = sorted(set(vals[i] for i in ids))
        ok = len(got) == 1
        check(ok, "T5 %s" % name, "%s -> %s" % (ids, got))
        print("%-8s %-28s %s => %s" % ("PASS" if ok else "FAIL", name, ids, got))

    # ---------- 6. 闸门自身的可证伪性（对整个 T1 闸门做反事实） ----------
    print()
    print("=" * 100)
    print("§6 闸门自身可证伪性：把「改后文档」换成「改前文档」，T1 闸门必须判红")
    print("=" * 100)
    red = 0
    green_expected = 0
    for (aid, path, lre, grp, cid, exempt, note) in CHANGED:
        rc, computed, err = r.sh(crit(cid))
        computed = int(computed)
        pre, _ = find_one(docs_base[path], lre, grp)
        verdict = "RED" if pre != computed else "GREEN"
        if exempt:
            green_expected += 1
            ok = (verdict == "GREEN")
        else:
            red += 1
            ok = (verdict == "RED")
        check(ok, "§6 gate-counterfactual %s" % aid,
              "改前文档值=%d 现算=%d ⇒ %s（豁免=%s）" % (pre, computed, verdict, exempt))
    print("%-8s 非豁免断言：改前文档判红 %d/%d；豁免断言（公共组件）：改前文档判绿 %d/%d" %
          ("PASS" if red == len(CHANGED) - green_expected else "FAIL",
           red, len(CHANGED) - green_expected, green_expected, green_expected))
    print("        ⇒ 结论：T1 闸门对 %d 条改值断言具备区分力（非恒真闸门）" % red)

    # ---------- 7. T6 已知残留（不判 FAIL，只登记） ----------
    print()
    print("=" * 100)
    print("§7 T6 已知残留核对（不判 FAIL，只核对事实）")
    print("=" * 100)
    ag = docs_tip["AGENTS.md"]

    # 7.1 模块表缺 News 行 / 无 chat_plus.go 覆盖
    has_news_row = bool(re.search(r"^\| \d+ \| \*\*News", ag, re.M))
    has_chatplus = bool(re.search(r"^\| \d+ \|.*chat_plus\.go", ag, re.M))
    check(not has_news_row and not has_chatplus, "T6 module-table-missing",
          "模块表 News 行=%s / chat_plus.go 覆盖=%s（内容列，父单 #1 不许改）" %
          (has_news_row, has_chatplus))
    print("%-8s 模块表缺 News 行=%s；无 chat_plus.go 模块行=%s（已知残留）" %
          ("PASS", not has_news_row, not has_chatplus))

    # 7.2 routes 树枚举的文件数 vs 实际
    tree = re.search(r"backend/routes/\n(.*?)```", ag, re.S)
    doc_routes = sorted(set(re.findall(r"[a-z0-9_]+\.go", tree.group(1)))) if tree else []
    _, actual_routes_raw, _ = r.sh("git ls-tree -r --name-only ${REF} -- backend/routes/ | grep '\\.go$'")
    actual_routes = sorted(set(x.split("/")[-1] for x in actual_routes_raw.splitlines() if x))
    missing_routes = [x for x in actual_routes if x not in doc_routes]
    check(len(missing_routes) > 0, "T6 routes-tree-enumeration",
          "routes 树枚举 %d 个 .go，实际 %d 个，缺 %s（已知残留）" %
          (len(doc_routes), len(actual_routes), missing_routes))
    print("%-8s routes 树枚举 %d 个 .go vs 实际 %d 个；树里缺：%s" %
          ("PASS", len(doc_routes), len(actual_routes), missing_routes))

    # 7.3 per-category 名字清单 vs 实际文件名（内容列残留）
    for cat in ("ai", "collab", "convert", "dev", "household", "life", "other", "share"):
        row = re.search(r"^├──? ?%s/.*$" % cat, ag, re.M)
        if not row:
            continue
        line = row.group(0)
        doc_names = []
        if "：" in line:
            doc_names = [x.strip().rstrip(",") for x in line.split("：", 1)[1].split(",")]
            doc_names = [x for x in doc_names if x and x != "..."]
        _, raw, _ = r.sh("git ls-tree -r --name-only ${REF} -- frontend/src/views/%s | grep '\\.vue$'" % cat)
        actual_names = sorted(set(x.split("/")[-1][:-4] for x in raw.splitlines() if x))
        unmatched_doc = [n for n in doc_names
                         if not any(n == a or a.startswith(n) or n.startswith(a) for a in actual_names)]
        unmatched_actual = [a for a in actual_names
                            if not any(n == a or a.startswith(n) or n.startswith(a) for n in doc_names)]
        check(True, "T6 category-names %s" % cat,
              "文档名 %d / 实际 %d；文档列了但无对应文件：%s；实际存在但文档未列：%s" %
              (len(doc_names), len(actual_names), unmatched_doc, unmatched_actual))
        flag = " ⇒ 名字清单不同步（内容列，另开单）" if (unmatched_doc or unmatched_actual) else ""
        print("%-8s category %-9s 文档名 %2d / 实际 %2d；文档多列=%s；实际未列=%s%s" %
              ("PASS", cat, len(doc_names), len(actual_names), unmatched_doc, unmatched_actual, flag))

    # ---------- 汇总 ----------
    print()
    print("=" * 100)
    print("汇总：%d 项检查，PASS %d，FAIL %d" %
          (len(rows), sum(1 for x in rows if x[0] == "PASS"), len(fails)))
    if fails:
        print("失败项：")
        for f in fails:
            print("  - " + f)
    print("=" * 100)
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
