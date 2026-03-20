"""
データ分析テンプレート (Python 3.13)
使い方: python3 template.py <csvファイルパス>
"""

import csv
import statistics
import json
import sys
from collections import Counter, defaultdict
from pathlib import Path


def load_csv(path: str) -> tuple[list[str], list[dict]]:
    """CSV を読み込んでヘッダーと行データを返す"""
    with open(path, encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f)
        headers = reader.fieldnames or []
        rows = list(reader)
    return list(headers), rows


def detect_numeric_columns(rows: list[dict], headers: list[str]) -> list[str]:
    """数値として解釈できる列を検出する"""
    numeric = []
    for col in headers:
        values = [r[col] for r in rows if r[col] != ""]
        if not values:
            continue
        try:
            [float(v) for v in values]
            numeric.append(col)
        except ValueError:
            pass
    return numeric


def basic_stats(rows: list[dict], col: str) -> dict:
    """1列の基本統計量を計算する"""
    values = [float(r[col]) for r in rows if r[col] != ""]
    if not values:
        return {"count": 0}

    sorted_vals = sorted(values)
    n = len(values)

    result = {
        "count": n,
        "mean": statistics.mean(values),
        "median": statistics.median(values),
        "min": sorted_vals[0],
        "max": sorted_vals[-1],
    }
    if n >= 2:
        result["stdev"] = statistics.stdev(values)
        result["variance"] = statistics.variance(values)

    # 四分位数
    q1_idx = n // 4
    q3_idx = (3 * n) // 4
    result["q1"] = sorted_vals[q1_idx]
    result["q3"] = sorted_vals[q3_idx]

    return result


def count_missing(rows: list[dict], headers: list[str]) -> dict[str, int]:
    """各列の欠損値数を返す"""
    missing: dict[str, int] = {}
    for col in headers:
        missing[col] = sum(1 for r in rows if r[col] == "")
    return missing


def value_counts(rows: list[dict], col: str, top_n: int = 10) -> list[tuple[str, int]]:
    """カテゴリ列の頻度集計"""
    counter = Counter(r[col] for r in rows if r[col] != "")
    return counter.most_common(top_n)


def correlation(rows: list[dict], col_a: str, col_b: str) -> float | None:
    """2列のピアソン相関係数を計算する"""
    pairs = [
        (float(r[col_a]), float(r[col_b]))
        for r in rows
        if r[col_a] != "" and r[col_b] != ""
    ]
    if len(pairs) < 2:
        return None

    xs = [p[0] for p in pairs]
    ys = [p[1] for p in pairs]
    n = len(pairs)
    mean_x = statistics.mean(xs)
    mean_y = statistics.mean(ys)

    numerator = sum((x - mean_x) * (y - mean_y) for x, y in pairs)
    denom_x = sum((x - mean_x) ** 2 for x in xs) ** 0.5
    denom_y = sum((y - mean_y) ** 2 for y in ys) ** 0.5

    if denom_x == 0 or denom_y == 0:
        return None
    return numerator / (denom_x * denom_y)


def generate_report(
    filepath: str,
    headers: list[str],
    rows: list[dict],
    numeric_cols: list[str],
    missing: dict[str, int],
) -> str:
    """Markdown レポートを生成する"""
    lines = ["# データ分析レポート", ""]

    # データ概要
    lines += [
        "## データ概要",
        f"- **ファイル名**: {Path(filepath).name}",
        f"- **総行数**: {len(rows):,}",
        f"- **列数**: {len(headers)}",
        f"- **列名**: {', '.join(headers)}",
        "",
        "### 欠損値",
        "| 列名 | 欠損数 | 欠損率 |",
        "|------|--------|--------|",
    ]
    for col in headers:
        m = missing[col]
        rate = m / len(rows) * 100 if rows else 0
        lines.append(f"| {col} | {m} | {rate:.1f}% |")
    lines.append("")

    # 基本統計
    if numeric_cols:
        lines += ["## 基本統計（数値列）", ""]
        for col in numeric_cols:
            stats = basic_stats(rows, col)
            lines += [
                f"### {col}",
                f"- 件数: {stats['count']:,}",
                f"- 平均: {stats.get('mean', 'N/A'):.4g}",
                f"- 中央値: {stats.get('median', 'N/A'):.4g}",
                f"- 標準偏差: {stats.get('stdev', 'N/A'):.4g}" if 'stdev' in stats else "- 標準偏差: N/A",
                f"- 最小: {stats.get('min', 'N/A'):.4g}",
                f"- Q1: {stats.get('q1', 'N/A'):.4g}",
                f"- Q3: {stats.get('q3', 'N/A'):.4g}",
                f"- 最大: {stats.get('max', 'N/A'):.4g}",
                "",
            ]

        # 相関行列
        if len(numeric_cols) >= 2:
            lines += ["## 相関行列", ""]
            header_row = "| 列名 | " + " | ".join(numeric_cols) + " |"
            separator = "|------|" + "--------|" * len(numeric_cols)
            lines += [header_row, separator]
            for col_a in numeric_cols:
                row_vals = [f"**{col_a}**"]
                for col_b in numeric_cols:
                    if col_a == col_b:
                        row_vals.append("1.000")
                    else:
                        corr = correlation(rows, col_a, col_b)
                        row_vals.append(f"{corr:.3f}" if corr is not None else "N/A")
                lines.append("| " + " | ".join(row_vals) + " |")
            lines.append("")

    # カテゴリ列の頻度
    category_cols = [c for c in headers if c not in numeric_cols]
    if category_cols:
        lines += ["## カテゴリ列の頻度（上位10件）", ""]
        for col in category_cols:
            counts = value_counts(rows, col)
            lines += [f"### {col}", "| 値 | 件数 |", "|----|----|"]
            for val, cnt in counts:
                lines.append(f"| {val} | {cnt} |")
            lines.append("")

    lines += ["## まとめ", "（エージェントが分析結果に基づいて記述します）", ""]
    return "\n".join(lines)


def main() -> None:
    if len(sys.argv) < 2:
        print("使い方: python3 template.py <csvファイルパス>", file=sys.stderr)
        sys.exit(1)

    filepath = sys.argv[1]
    headers, rows = load_csv(filepath)
    numeric_cols = detect_numeric_columns(rows, headers)
    missing = count_missing(rows, headers)

    report = generate_report(filepath, headers, rows, numeric_cols, missing)

    output_path = Path("report.md")
    output_path.write_text(report, encoding="utf-8")
    print(f"✅ レポートを {output_path} に書き出しました")
    print(f"   行数: {len(rows):,}, 列数: {len(headers)}, 数値列: {numeric_cols}")


if __name__ == "__main__":
    main()
