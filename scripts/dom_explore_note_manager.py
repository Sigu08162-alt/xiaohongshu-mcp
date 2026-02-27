#!/usr/bin/env python3
"""
DOM 探索脚本 - 打印 note-manager 页面的笔记列表结构
用途：找到删除按钮的真实 selector，不做任何写操作
"""
import asyncio
import json
import sys
from pathlib import Path
from playwright.async_api import async_playwright

COOKIES_PATH = Path(__file__).parent.parent / "cookies.json"
SCREENSHOT_PATH = "/tmp/note_manager_dom.png"

async def main():
    if not COOKIES_PATH.exists():
        print(f"❌ cookies.json 不存在: {COOKIES_PATH}")
        sys.exit(1)

    with open(COOKIES_PATH) as f:
        cookies = json.load(f)

    async with async_playwright() as p:
        browser = await p.chromium.launch(
            headless=True,
            args=["--no-sandbox", "--disable-setuid-sandbox"]
        )
        context = await browser.new_context(
            viewport={"width": 1920, "height": 1080},
            user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
        )

        # 注入 cookies
        await context.add_cookies(cookies)

        page = await context.new_page()

        print("🔍 先访问主站，再跳转到创作者中心...")
        try:
            await page.goto("https://www.xiaohongshu.com", wait_until="domcontentloaded", timeout=30000)
        except Exception as e:
            print(f"⚠️ 主站加载警告: {e}")
        await asyncio.sleep(2)

        print("🔍 导航到创作者中心笔记管理页...")
        try:
            await page.goto("https://creator.xiaohongshu.com/new/note-manager", wait_until="commit", timeout=60000)
        except Exception as e:
            print(f"⚠️ 导航警告（继续）: {e}")
        await asyncio.sleep(3)

        # 截图
        await page.screenshot(path=SCREENSHOT_PATH, full_page=False)
        print(f"📸 截图已保存: {SCREENSHOT_PATH}")

        # 当前 URL（检查是否被重定向到验证码）
        current_url = page.url
        print(f"📍 当前 URL: {current_url}")

        if "captcha" in current_url or "login" in current_url:
            print("❌ 被重定向到验证码/登录页，cookie 已失效")
            await browser.close()
            return

        # 打印页面标题
        title = await page.title()
        print(f"📄 页面标题: {title}")

        # 打印笔记卡片区域 HTML（前5000字符）
        html = await page.evaluate("() => document.body.innerHTML.slice(0, 5000)")
        print(f"\n📦 页面 HTML（前5000字符）:\n{html}\n")

        # 查找所有卡片/列表元素
        cards = await page.evaluate("""() => {
            const selectors = [
                '.note-item', '.note-card', '.content-item', '.publish-item',
                '[class*="note-item"]', '[class*="note-card"]', '[class*="content-item"]',
                '[class*="note-list"]', '[class*="card-wrap"]', 'li[class*="note"]',
                'div[class*="item-wrap"]', 'div[class*="note-wrap"]'
            ];
            const results = [];
            for (const sel of selectors) {
                const els = document.querySelectorAll(sel);
                if (els.length > 0) {
                    results.push(`✅ ${sel}: ${els.length}个元素, class="${els[0].className}"`);
                }
            }
            return results.join('\\n') || '❌ 未找到任何笔记卡片元素';
        }""")
        print(f"\n🃏 笔记卡片元素:\n{cards}\n")

        # 查找所有按钮
        buttons = await page.evaluate("""() => {
            const els = [...document.querySelectorAll('button, [class*="btn"], [class*="button"], [class*="more"], [class*="operate"], [class*="action"], [class*="delete"]')];
            return els.slice(0, 50).map(e => `${e.tagName} class="${e.className}" text="${e.innerText.trim().slice(0,30)}" visible=${e.offsetParent !== null}`).join('\\n');
        }""")
        print(f"\n🔘 所有按钮元素（前50个）:\n{buttons}\n")

        # 尝试 hover 第一个笔记卡片，看是否出现操作按钮
        print("🖱️ 尝试 hover 第一个笔记相关元素...")
        hovered = await page.evaluate("""async () => {
            const selectors = [
                '.note-item', '[class*="note-item"]', '[class*="note-card"]',
                '[class*="content-item"]', '[class*="card-wrap"]', 'li'
            ];
            for (const sel of selectors) {
                const el = document.querySelector(sel);
                if (el) {
                    el.dispatchEvent(new MouseEvent('mouseenter', {bubbles: true}));
                    el.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}));
                    return `✅ hover: ${sel} class="${el.className}"`;
                }
            }
            return '❌ 没有找到可 hover 的元素';
        }""")
        print(f"hover 结果: {hovered}")
        await asyncio.sleep(1)

        # hover 后再打印按钮
        buttons_after = await page.evaluate("""() => {
            const els = [...document.querySelectorAll('button, [class*="btn"], [class*="button"], [class*="more"], [class*="operate"], [class*="action"], [class*="delete"]')];
            return els.slice(0, 50).map(e => `${e.tagName} class="${e.className}" text="${e.innerText.trim().slice(0,30)}" visible=${e.offsetParent !== null}`).join('\\n');
        }""")
        print(f"\n🔘 hover 后按钮元素:\n{buttons_after}\n")

        await page.screenshot(path="/tmp/note_manager_hover.png")
        print("📸 hover 后截图: /tmp/note_manager_hover.png")

        await browser.close()
        print("\n✅ DOM 探索完成")

if __name__ == "__main__":
    asyncio.run(main())
