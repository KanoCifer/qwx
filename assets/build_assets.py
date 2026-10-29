#!/usr/bin/env python3
"""生成 README 用的 logo 与横幅。

图标源图是手绘的 PNG（assets/emblem-source.png），本脚本只做确定性加工：
排版文字、圆角裁切，最后交给 rsvg-convert 渲染成 PNG。
中间 SVG 写在临时目录，不进仓库。

依赖：rsvg-convert（brew install librsvg）。
"""

import base64
import pathlib
import shutil
import subprocess
import tempfile

ASSETS = pathlib.Path(__file__).resolve().parent
SOURCE = ASSETS / "qwx.png"
TMP = pathlib.Path(tempfile.mkdtemp(prefix="qwx-assets-"))


def source_data_uri() -> str:
    raw = SOURCE.read_bytes()
    return "data:image/png;base64," + base64.b64encode(raw).decode()


def render(svg: str, dst: str, width: int) -> None:
    svg_path = TMP / f"{dst}.svg"
    svg_path.write_text(svg)
    subprocess.run(
        ["rsvg-convert", "-w", str(width), "-o", str(ASSETS / dst), str(svg_path)],
        check=True,
    )


def write_logo() -> None:
    shutil.copyfile(SOURCE, ASSETS / "logo.png")


def banner_svg(uri: str, subtitle: str, subtitle_font: str) -> str:
    return f"""<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="1280" height="320" viewBox="0 0 1280 320">
  <title>qwx</title>
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#062F4E" />
      <stop offset="55%" stop-color="#0B6FA4" />
      <stop offset="100%" stop-color="#00ADD8" />
    </linearGradient>
    <clipPath id="iconClip">
      <rect x="40" y="32" width="256" height="256" rx="52" />
    </clipPath>
  </defs>
  <rect x="0" y="0" width="1280" height="320" rx="18" fill="url(#bg)" />
  <circle cx="1215" cy="55" r="150" fill="#FFFFFF" opacity="0.06" />
  <circle cx="1150" cy="290" r="90" fill="#FFFFFF" opacity="0.05" />
  <circle cx="90" cy="300" r="120" fill="#FFFFFF" opacity="0.04" />
  <g clip-path="url(#iconClip)">
    <rect x="40" y="32" width="256" height="256" fill="#FFFFFF" />
    <image xlink:href="{uri}" x="40" y="32" width="256" height="256" />
  </g>
  <text x="336" y="170" font-family="Helvetica Neue, Helvetica, Arial, sans-serif" font-size="104" font-weight="700" fill="#FFFFFF" letter-spacing="-2">qwx</text>
  <text x="340" y="222" font-family="Helvetica Neue, Helvetica, Arial, sans-serif" font-size="30" font-weight="600" fill="#9FE9FF">Go SDK for QWeather</text>
  <text x="340" y="264" font-family="{subtitle_font}" font-size="26" fill="#D5F2FD">{subtitle}</text>
</svg>
"""


def write_banner() -> None:
    render(
        banner_svg(
            source_data_uri(),
            "和风天气开发服务的 Go 客户端 · 核心包零第三方依赖",
            "PingFang SC, Hiragino Sans GB, Helvetica Neue, Helvetica, sans-serif",
        ),
        "banner.png",
        1280,
    )


def write_banner_en() -> None:
    render(
        banner_svg(
            source_data_uri(),
            "Typed responses · Zero-dependency core · Every core endpoint family",
            "Helvetica Neue, Helvetica, Arial, sans-serif",
        ),
        "banner_en.png",
        1280,
    )


if __name__ == "__main__":
    write_logo()
    write_banner()
    write_banner_en()
    print("done")
