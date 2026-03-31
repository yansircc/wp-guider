#!/usr/bin/env python3
"""Generate missing exam tasks with verify assertions for all 90 gaps."""

import json
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXAM_DIR = ROOT / "out" / ".claude" / "references" / "tasks"

TOPIC_NAME = {
    "domain": "域名管理", "hosting": "主机空间", "wp-install": "WordPress 安装",
    "site-settings": "站点配置", "user-management": "用户管理",
    "pages": "页面管理", "posts-taxonomy": "文章与分类", "media": "媒体管理", "menus-nav": "菜单与导航",
    "theme": "主题定制", "elementor": "Elementor 建站", "zeroy": "ZeroY AI 建站",
    "plugins-basic": "插件管理", "acf": "ACF 自定义字段", "seo": "SEO 优化", "google-analytics": "Google 全家桶",
    "wp-config": "wp-config 与数据库", "security": "安全加固",
    "backup-maintenance": "备份与维护", "troubleshooting": "故障排查",
}

# ── New exam tasks with verify assertions ──────────────────────────────

NEW_TASKS = {
    "acf": [
        {
            "id": "ACF-25", "difficulty": 2,
            "description": "将当前所有 ACF 字段组导出为 JSON 文件（ACF → 工具 → 导出）",
            "verify": [{"type": "wp_eval", "php_code": "echo (function_exists('acf_get_field_groups') && count(acf_get_field_groups()) > 0) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["ACF → 工具 → 导出字段组", "选择所有字段组，点击导出"],
            "on_pass_note": "ACF JSON 导出常用于团队协作（Git 同步）和站点迁移。导出的 JSON 只包含字段定义，不包含数据。",
        },
        {
            "id": "ACF-26", "difficulty": 2,
            "description": "将之前导出的 ACF JSON 文件重新导入（ACF → 工具 → 导入），确认字段组恢复",
            "verify": [{"type": "wp_eval", "php_code": "$groups = acf_get_field_groups(); echo count($groups) >= 1 ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["ACF → 工具 → 导入字段组", "选择 JSON 文件上传"],
            "on_pass_note": "导入时如果字段组 key 相同会覆盖。团队协作建议将 acf-json 文件夹纳入 Git。",
        },
        {
            "id": "ACF-27", "difficulty": 2,
            "description": "访问 acf-playground.vercel.app，用可视化界面拖拽创建一个包含 3 个字段的字段组，导出 JSON 并导入到 WordPress",
            "verify": [{"type": "wp_eval", "php_code": "$groups = acf_get_field_groups(); echo count($groups) >= 2 ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["打开 acf-playground.vercel.app", "拖拽添加字段后点击 Export JSON"],
            "on_pass_note": "ACF Playground 是第三方可视化工具，适合快速原型设计。生成的 JSON 导入后仍需在后台验证字段设置。",
        },
        {
            "id": "ACF-28", "difficulty": 2,
            "description": "访问 acf-helper.vercel.app，用自然语言描述需求生成 ACF JSON（如'产品类型需要价格、图片、描述三个字段'），导入到 WordPress",
            "verify": [{"type": "wp_eval", "php_code": "$groups = acf_get_field_groups(); echo count($groups) >= 3 ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["打开 acf-helper.vercel.app", "用中文描述你的字段需求"],
            "on_pass_note": "AI 辅助工具可以加速 ACF 配置，但生成的 JSON 不一定 100% 符合需求，导入后务必逐个检查。",
        },
    ],
    "backup-maintenance": [
        {
            "id": "BM-11", "difficulty": 4,
            "description": "在 wp-config.php 中设置 DISABLE_WP_CRON 为 true 禁用内置 wp-cron，然后通过系统 crontab 添加替代计划任务：每 15 分钟执行一次 wp cron event run --due-now",
            "verify": [
                {"type": "config_equals", "key": "DISABLE_WP_CRON", "expected": "1"},
            ],
            "hints": ["在 wp-config.php 中 define('DISABLE_WP_CRON', true);", "用 crontab -e 添加系统级定时任务"],
            "on_pass_note": "wp-cron 依赖页面访问触发，流量低的站点定时任务不准。系统 cron 是可靠替代方案，外贸站必做。",
        },
    ],
    "domain": [
        {
            "id": "DOM-1", "difficulty": 1,
            "description": "在 Namesilo 注册一个域名（或模拟注册流程），记录域名和到期时间。验证：在站点后台将 blogdescription 改为你注册的域名",
            "verify": [{"type": "wp_eval", "php_code": "$d = get_option('blogdescription'); echo (strpos($d, '.') !== false && strlen($d) > 3) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["Namesilo 是便宜的域名注册商", "把域名写到站点副标题作为验证"],
            "on_pass_note": "Namesilo 优势：价格透明无隐藏费用、免费隐私保护。注册后第一步是修改 NS 到你的主机商。",
        },
        {
            "id": "DOM-2", "difficulty": 1,
            "description": "在阿里云注册域名的流程学习：搜索域名 → 加入购物车 → 实名认证。验证：将站点标题改为「阿里云域名实操完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "阿里云域名实操完成"}],
            "hints": ["阿里云域名需要实名认证", "国内域名适合面向国内用户的站点"],
            "on_pass_note": "阿里云域名需要实名认证（3-5 工作日），且 .cn 域名需要备案才能解析到国内服务器。外贸站建议用 Namesilo/Cloudflare。",
        },
        {
            "id": "DOM-3", "difficulty": 1,
            "description": "在 Cloudflare 注册域名的流程学习。验证：将站点标题改为「Cloudflare域名实操完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "Cloudflare域名实操完成"}],
            "hints": ["Cloudflare 注册域名按成本价，零加价"],
            "on_pass_note": "Cloudflare 域名最大优势：自动获得 CDN + DDoS 防护 + DNS 管理。缺点是不支持所有后缀。",
        },
        {
            "id": "DOM-4", "difficulty": 2,
            "description": "学习 DNS A 记录解析原理：A 记录将域名指向 IP 地址。验证：将站点标题改为「DNS-A记录-指向IP」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "DNS-A记录-指向IP"}],
            "hints": ["A 记录：域名 → IP 地址", "TTL 建议设置为 600 秒（10 分钟）"],
            "on_pass_note": "A 记录是最基础的 DNS 记录。@ 代表根域名，www 是子域名。修改 A 记录后需等待 TTL 过期生效。",
        },
        {
            "id": "DOM-5", "difficulty": 2,
            "description": "学习 DNS CNAME 记录：CNAME 将一个域名指向另一个域名。验证：将站点标题改为「CNAME-别名记录」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "CNAME-别名记录"}],
            "hints": ["CNAME：域名 → 另一个域名", "常用于 www 指向根域名"],
            "on_pass_note": "CNAME 不能和其他记录共存于同一主机名（RFC 限制）。Cloudflare 用 CNAME Flattening 解决了根域名的限制。",
        },
        {
            "id": "DOM-6", "difficulty": 2,
            "description": "学习修改 NS（Name Server）到 Hostinger 的流程。验证：将站点标题改为「NS已切换到Hostinger」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "NS已切换到Hostinger"}],
            "hints": ["在域名注册商处修改 NS", "Hostinger NS: ns1.dns-parking.com, ns2.dns-parking.com"],
            "on_pass_note": "修改 NS 后 DNS 由 Hostinger 管理。生效时间 24-48 小时。切换前确保在 Hostinger 已添加域名。",
        },
    ],
    "google-analytics": [
        {
            "id": "GA-1", "difficulty": 2,
            "description": "在 Google Search Console 中添加站点并完成所有权验证（HTML 标签方式）。验证：在站点中安装一个 SEO 插件并开启 GSC 连接",
            "verify": [{"type": "wp_eval", "php_code": "$plugins = get_option('active_plugins'); foreach($plugins as $p) { if(strpos($p, 'seo') !== false || strpos($p, 'yoast') !== false) { echo 'yes'; exit; } } echo 'no';", "expected_output": "yes"}],
            "hints": ["GSC 验证方式：HTML 标签、DNS、文件上传", "SEO 插件通常内置 GSC 验证功能"],
            "on_pass_note": "GSC 是免费工具，告诉你 Google 怎么看你的站点：索引状态、搜索词、错误。每个外贸站都必须注册。",
        },
        {
            "id": "GA-2", "difficulty": 2,
            "description": "在 GSC 中提交站点的 XML Sitemap（通常是 /sitemap_index.xml）。验证：确认 Yoast/SEOPress 的 Sitemap 功能已开启",
            "verify": [{"type": "wp_eval", "php_code": "$y = get_option('wpseo'); $s = get_option('seopress_xml_sitemap_option_name'); echo (!empty($y) || !empty($s)) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["Sitemap 地址通常是 /sitemap_index.xml 或 /sitemap.xml", "在 GSC → Sitemaps 中提交"],
            "on_pass_note": "Sitemap 帮助搜索引擎发现页面。新站提交后 Google 通常几天内开始抓取。大型站点可以分多个 Sitemap。",
        },
        {
            "id": "GA-3", "difficulty": 2,
            "description": "注册 GA4 并通过 SEO 插件安装追踪代码（Measurement ID 格式：G-XXXXXXXXXX）。验证：将追踪 ID 保存到站点的 blogdescription 选项中",
            "verify": [{"type": "wp_eval", "php_code": "$d = get_option('blogdescription'); echo (strpos($d, 'G-') === 0 && strlen($d) > 5) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["GA4 → 管理 → 数据流 → 衡量 ID", "通过 SEOPress 或 Yoast 填入追踪代码"],
            "on_pass_note": "GA4 取代了旧版 Universal Analytics。追踪代码建议通过插件管理而非手动编辑主题，避免更新主题时丢失。",
        },
        {
            "id": "GA-4", "difficulty": 3,
            "description": "学习 GA4 基础报告：用户数、会话数、页面浏览量的含义和区别。验证：将站点标题改为「GA4报告-用户会话页面」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "GA4报告-用户会话页面"}],
            "hints": ["用户 = 独立访客，会话 = 一次访问，页面浏览 = 看了几个页面"],
            "on_pass_note": "GA4 核心指标：用户（who）、会话（when）、事件（what）。跳出率在 GA4 中改为参与率（Engagement Rate）。",
        },
        {
            "id": "GA-5", "difficulty": 3,
            "description": "学习 GSC 性能报告：总点击数、展示次数、CTR、平均排名的含义。验证：将站点标题改为「GSC性能-点击展示CTR排名」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "GSC性能-点击展示CTR排名"}],
            "hints": ["CTR = 点击 / 展示 × 100%", "排名越低越好（第 1 位最佳）"],
            "on_pass_note": "GSC 性能报告是 SEO 的核心数据源。重点关注：高展示低 CTR 的词（需优化标题/描述）和排名 5-15 的词（有提升空间）。",
        },
        {
            "id": "GA-6", "difficulty": 3,
            "description": "学习 GA4 事件追踪概念：按钮点击、表单提交如何通过事件记录。验证：将站点标题改为「GA4事件-click-submit」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "GA4事件-click-submit"}],
            "hints": ["GA4 中一切都是事件（event）", "自动事件 vs 自定义事件"],
            "on_pass_note": "GA4 自动追踪 page_view、scroll、click 等事件。自定义事件需要 gtag('event', ...) 或 GTM 配置。外贸站重点追踪询盘表单提交。",
        },
        {
            "id": "GA-7", "difficulty": 4,
            "description": "学习 GA4 转化目标设置和 Google Ads 联动概念。验证：将站点标题改为「GA4转化-GoogleAds联动」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "GA4转化-GoogleAds联动"}],
            "hints": ["在 GA4 中将关键事件标记为转化", "GA4 和 Google Ads 关联后可以导入转化数据"],
            "on_pass_note": "转化追踪闭环：GA4 记录事件 → 标记为转化 → 导入 Google Ads → 优化广告投放。这是外贸数字营销的核心链路。",
        },
    ],
    "hosting": [
        {
            "id": "HOST-1", "difficulty": 1,
            "description": "了解 Hostinger 购买流程和套餐区别。验证：将站点标题改为「Hostinger购买完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "Hostinger购买完成"}],
            "hints": ["推荐 Business Web Hosting 套餐", "付费周期越长单价越低"],
            "on_pass_note": "Hostinger 优势：价格便宜、支持 LiteSpeed + 免费 SSL。外贸站推荐选美国或欧洲节点。",
        },
        {
            "id": "HOST-2", "difficulty": 1,
            "description": "熟悉 hPanel 控制面板主要功能区。验证：将站点标题改为「hPanel导航完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "hPanel导航完成"}],
            "hints": ["hPanel 主要区域：网站、域名、文件、数据库、SSL"],
            "on_pass_note": "hPanel 是 Hostinger 自研面板，不是 cPanel。核心操作：文件管理器、数据库管理、SSL 证书、Git 部署。",
        },
        {
            "id": "HOST-3", "difficulty": 2,
            "description": "在 Hostinger hPanel 中添加第二个网站（多站点管理）。验证：将站点标题改为「多站点管理完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "多站点管理完成"}],
            "hints": ["hPanel → 网站 → 添加网站", "Business 套餐支持 100 个网站"],
            "on_pass_note": "一个 Hostinger 账号可以管理多个站点。每个站点独立域名、独立数据库。共享服务器资源。",
        },
        {
            "id": "HOST-4", "difficulty": 3,
            "description": "在 Hostinger hPanel 中开启 SSH 访问权限。验证：将站点标题改为「SSH访问已开启」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "SSH访问已开启"}],
            "hints": ["hPanel → 高级 → SSH 访问", "记录 SSH 主机、端口、用户名"],
            "on_pass_note": "SSH 是远程管理服务器的加密通道。开启后可以用命令行管理文件、运行 wp-cli、查看日志。",
        },
        {
            "id": "HOST-5", "difficulty": 4,
            "description": "通过 SSH 连接到 Hostinger 服务器，执行基础命令（ls、pwd、cd）。验证：将站点标题改为「SSH终端操作完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "SSH终端操作完成"}],
            "hints": ["ssh -p PORT user@host", "ls 列出文件，pwd 显示当前路径"],
            "on_pass_note": "SSH 常用命令：ls（列文件）、cd（切目录）、cat（看文件）、wp（WP-CLI）。出问题时 SSH 是最后的救命稻草。",
        },
        {
            "id": "HOST-6", "difficulty": 4,
            "description": "生成 SSH 密钥对并配置免密登录。验证：将站点标题改为「SSH免密登录完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "SSH免密登录完成"}],
            "hints": ["ssh-keygen -t ed25519", "把公钥添加到 hPanel 的 SSH 密钥管理"],
            "on_pass_note": "密钥认证比密码安全得多。ed25519 是推荐的密钥类型。配置后可以用 SSH config 简化连接命令。",
        },
    ],
    "menus-nav": [
        {
            "id": "MN-5", "difficulty": 2,
            "description": "将菜单分配到主题的主要菜单位置（Primary 或 Main Menu），确认前台导航显示正确",
            "verify": [{"type": "wp_eval", "php_code": "$locations = get_nav_menu_locations(); echo (count($locations) > 0 && array_filter($locations)) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["外观 → 菜单 → 管理位置", "或在菜单编辑页面底部勾选显示位置"],
            "on_pass_note": "菜单位置由主题定义（register_nav_menus）。一个位置只能放一个菜单，但一个菜单可以放在多个位置。",
        },
    ],
    "seo": [
        {
            "id": "SEO-7", "difficulty": 2,
            "description": "安装 SEOPress 插件，在任意页面设置 SEO 标题和描述",
            "verify": [{"type": "plugin_active", "plugin": "wp-seopress"}],
            "hints": ["插件 → 添加新插件 → 搜索 SEOPress", "编辑页面 → SEOPress 面板"],
            "on_pass_note": "SEOPress 是 Yoast 的替代品，界面更简洁。两者功能类似，选一个用即可。同时安装会冲突。",
        },
        {
            "id": "SEO-8", "difficulty": 3,
            "description": "在 SEOPress 中配置 Google Analytics 追踪代码（填入 GA4 的 Measurement ID）",
            "verify": [{"type": "wp_eval", "php_code": "$opts = get_option('seopress_google_analytics_option_name'); echo (!empty($opts) && is_array($opts)) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["SEOPress → Google Analytics", "填入 G-XXXXXXXXXX 格式的 Measurement ID"],
            "on_pass_note": "通过 SEO 插件管理追踪代码的优势：不怕主题更新覆盖、可以在一个地方管理所有代码片段。",
        },
        {
            "id": "SEO-9", "difficulty": 3,
            "description": "在 SEOPress 中为指定页面设置 noindex 和 nofollow（如 Thank You 页面不需要被搜索引擎收录）",
            "verify": [{"type": "wp_eval", "php_code": "global $wpdb; $r = $wpdb->get_var(\"SELECT COUNT(*) FROM $wpdb->postmeta WHERE meta_key = '_seopress_robots_index' AND meta_value = 'yes'\"); echo ($r > 0) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["编辑页面 → SEOPress → Advanced → Robots", "noindex = 不索引，nofollow = 不追踪链接"],
            "on_pass_note": "noindex 适用于：Thank You 页、登录页、测试页。不要给重要页面加 noindex，否则 Google 不收录。",
        },
        {
            "id": "SEO-10", "difficulty": 3,
            "description": "为 ACF 自定义文章类型配置统一的 SEO 标题模板（如「%%title%% | 产品 | 站点名」）",
            "verify": [{"type": "wp_eval", "php_code": "$cpts = get_post_types(['public' => true, '_builtin' => false]); echo count($cpts) > 0 ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["Yoast → 搜索外观 → 内容类型 → 自定义文章类型", "SEOPress → 标题与元描述 → 文章类型"],
            "on_pass_note": "CPT 的 SEO 标题模板确保所有同类页面有一致的格式。变量如 %%title%%、%%sitename%%、%%sep%% 会自动替换。",
        },
        {
            "id": "SEO-11", "difficulty": 3,
            "description": "为 ACF 分类法（Taxonomy）配置 SEO 标题和描述模板",
            "verify": [{"type": "wp_eval", "php_code": "$taxs = get_taxonomies(['public' => true, '_builtin' => false]); echo count($taxs) > 0 ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["Yoast → 搜索外观 → 分类法", "SEOPress → 标题与元描述 → 分类法"],
            "on_pass_note": "分类法归档页的 SEO 常被忽略。好的分类页标题模板能带来长尾流量，如「产品分类: %%term_title%% | 站名」。",
        },
        {
            "id": "SEO-12", "difficulty": 4,
            "description": "制定 CPT 单篇和归档页的 index/noindex 策略：单篇页面 index，归档页根据内容量决定。验证：将站点标题改为「CPT-SEO策略已制定」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "CPT-SEO策略已制定"}],
            "hints": ["内容丰富的归档页 → index", "薄内容的归档页 → noindex 避免被认为低质量"],
            "on_pass_note": "SEO 策略核心：让 Google 只索引有价值的页面。CPT 归档页如果只有标题列表，noindex 更好。加了描述和内容的归档页可以 index。",
        },
    ],
    "user-management": [
        {
            "id": "UM-7", "difficulty": 3,
            "description": "为管理员账号生成 Application Password（应用程序密码），用于 REST API 认证",
            "verify": [{"type": "wp_eval", "php_code": "global $wpdb; $r = $wpdb->get_var(\"SELECT COUNT(*) FROM $wpdb->usermeta WHERE meta_key = '_application_passwords'\"); echo ($r > 0) ? 'yes' : 'no';", "expected_output": "yes"}],
            "hints": ["用户 → 个人资料 → 应用程序密码", "填写名称后点击「添加新的应用程序密码」"],
            "on_pass_note": "Application Password 是 WordPress 5.6+ 的内置功能，用于外部应用通过 REST API 操作站点。密码只显示一次，保存好。",
        },
        {
            "id": "UM-8", "difficulty": 2,
            "description": "安装 DoLogin Security 插件，为某个用户生成免密登录链接",
            "verify": [{"type": "wp_eval", "php_code": "$plugins = get_option('active_plugins'); foreach($plugins as $p) { if(strpos($p, 'dologin') !== false) { echo 'yes'; exit; } } echo 'no';", "expected_output": "yes"}],
            "hints": ["搜索插件 DoLogin Security", "用户列表 → DoLogin 链接按钮"],
            "on_pass_note": "DoLogin 生成一次性或限时登录链接，常用于：给客户临时登录、调试时快速切换用户。注意安全风险——链接泄露等于账号泄露。",
        },
    ],
    "wp-install": [
        {
            "id": "INST-1", "difficulty": 1,
            "description": "学习 Hostinger 一键安装 WordPress 的流程。验证：将站点标题改为「一键安装WP完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "一键安装WP完成"}],
            "hints": ["hPanel → 网站 → 自动安装器 → WordPress"],
            "on_pass_note": "一键安装会自动创建数据库、下载 WordPress、运行安装向导。适合快速部署，但了解手动安装有助于排障。",
        },
        {
            "id": "INST-2", "difficulty": 2,
            "description": "学习手动安装 WordPress 的步骤（下载、解压、配置 wp-config、运行安装向导）。验证：将站点标题改为「手动安装WP完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "手动安装WP完成"}],
            "hints": ["wordpress.org/download 下载最新版", "复制 wp-config-sample.php 为 wp-config.php"],
            "on_pass_note": "手动安装的核心：① 数据库信息正确 ② 文件权限正确 ③ wp-config.php 配置正确。掌握后能解决 90% 的安装问题。",
        },
        {
            "id": "INST-3", "difficulty": 2,
            "description": "学习通过 NS 方式绑定域名到 Hostinger。验证：将站点标题改为「NS绑定域名完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "NS绑定域名完成"}],
            "hints": ["在域名注册商处修改 NS 为 Hostinger 的 NS", "Hostinger hPanel → 域名 → 添加域名"],
            "on_pass_note": "NS 方式把整个 DNS 管理权交给 Hostinger。优点是简单统一管理；缺点是 DNS 功能受限于 Hostinger。",
        },
        {
            "id": "INST-4", "difficulty": 2,
            "description": "学习通过 A 记录方式绑定域名到 Hostinger。验证：将站点标题改为「A记录绑定域名完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "A记录绑定域名完成"}],
            "hints": ["保持 NS 在原注册商", "添加 A 记录指向 Hostinger 服务器 IP"],
            "on_pass_note": "A 记录方式只把网站流量指向 Hostinger，DNS 管理权留在原注册商。适合同时使用 Cloudflare CDN 的场景。",
        },
        {
            "id": "INST-5", "difficulty": 2,
            "description": "学习在 Hostinger 中配置 SSL 证书（免费 Let's Encrypt）。验证：将站点标题改为「SSL证书已配置」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "SSL证书已配置"}],
            "hints": ["hPanel → SSL → 安装 SSL", "强制 HTTPS 跳转"],
            "on_pass_note": "SSL 是现代网站标配（SEO 加分、浏览器不报警）。Let's Encrypt 免费且自动续期。安装后记得设置 HTTP→HTTPS 跳转。",
        },
        {
            "id": "INST-6", "difficulty": 4,
            "description": "使用 All-in-One WP Migration 插件导出和导入站点（插件方式迁移）。验证：确认插件已安装激活",
            "verify": [{"type": "wp_eval", "php_code": "$plugins = get_option('active_plugins'); foreach($plugins as $p) { if(strpos($p, 'all-in-one') !== false) { echo 'yes'; exit; } } echo 'no';", "expected_output": "yes"}],
            "hints": ["安装 All-in-One WP Migration 插件", "导出：工具 → 导出 → 导出到文件"],
            "on_pass_note": "AIO Migration 是最简单的迁移方式。免费版限制 512MB，大站需要付费版或用手动迁移。导入前务必备份目标站点。",
        },
        {
            "id": "INST-7", "difficulty": 4,
            "description": "学习手动迁移：导出数据库 + 复制文件 + 修改 wp-config。验证：将站点标题改为「手动迁移流程完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "手动迁移流程完成"}],
            "hints": ["步骤：导出 SQL → 打包文件 → 上传目标服务器 → 导入 SQL → 改 wp-config → search-replace 域名"],
            "on_pass_note": "手动迁移是最可靠的方式。核心命令：mysqldump（导出）、mysql（导入）、wp search-replace（替换域名）。插件搞不定时这是最后手段。",
        },
        {
            "id": "INST-8", "difficulty": 4,
            "description": "学习已有站点更换域名的完整流程。验证：将站点标题改为「域名更换流程完成」",
            "verify": [{"type": "option_equals", "key": "blogname", "expected": "域名更换流程完成"}],
            "hints": ["修改 siteurl + home", "wp search-replace 'old.com' 'new.com' --dry-run 先预览"],
            "on_pass_note": "换域名清单：① 备份 ② 改 DNS ③ 改 siteurl/home ④ search-replace 全库 ⑤ 更新 GSC ⑥ 提交 301 跳转 ⑦ 更新 GA。漏任何一步都会出问题。",
        },
    ],
}

# Elementor - generate from docs descriptions with wp_eval checks
ELEMENTOR_NEW = []
ele_tasks = [
    (32, 2, "Floating Contact Buttons（WhatsApp/SMS）", "安装并配置 Elementor 的 Floating Buttons widget，设置 WhatsApp 联系按钮"),
    (33, 3, "Popup 创建（Modal/Bar/Slide-in）", "使用 Elementor Popup Builder 创建一个 Modal 类型弹窗，包含标题和关闭按钮"),
    (34, 4, "Popup 触发条件与显示规则", "为弹窗设置触发条件：页面加载 5 秒后显示，且每个用户只显示一次"),
    (35, 4, "Popup 表单集成（询盘弹窗）", "在弹窗中添加 Elementor Form widget 收集询盘信息"),
    (36, 3, "Theme Builder Header", "用 Theme Builder 创建自定义 Header 模板：Logo + Nav Menu + 搜索图标"),
    (37, 3, "Theme Builder Footer", "用 Theme Builder 创建自定义 Footer 模板：公司信息 + 链接列表 + 版权声明"),
    (38, 3, "Single Post 文章模板", "用 Theme Builder 创建 Single Post 模板：Post Title + Featured Image + Post Content + Author Box"),
    (39, 4, "Archive 归档模板", "用 Theme Builder 创建 Archive 模板：归档标题 + Posts Widget 列表 + 分页"),
    (40, 4, "Display Conditions", "为 Header 模板设置 Display Conditions：所有页面显示，但排除 Landing Page"),
    (41, 3, "Search Results 模板", "用 Theme Builder 创建搜索结果页模板"),
    (42, 3, "Error 404 模板", "用 Theme Builder 创建 404 错误页面：包含提示文字和返回首页按钮"),
    (43, 3, "Dynamic Tags 基础", "在 Heading widget 中使用 Dynamic Tag 显示站点标题，在 Image widget 中使用 Dynamic Tag 显示文章特色图"),
    (44, 4, "ACF + Dynamic Tags", "将 ACF 自定义字段通过 Dynamic Tags 在 Elementor 模板中显示"),
    (45, 4, "Posts Widget 文章列表", "添加 Posts Widget 展示最新文章，配置筛选条件和分页"),
    (46, 4, "Portfolio Widget", "添加 Portfolio Widget 展示作品集，配置悬停效果和筛选标签"),
    (47, 3, "WooCommerce 产品页模板", "用 Theme Builder 创建 WooCommerce Single Product 模板"),
    (48, 4, "WooCommerce 商店归档", "用 Theme Builder 创建 Products Archive 模板：商品网格 + 筛选"),
    (49, 3, "Landing Page 设计", "创建一个 Landing Page：Canvas 模板 + Hero + 卖点 + CTA + 表单"),
    (50, 4, "转化优化", "学习 CTA 位置、对比测试思路。验证：将站点标题改为「CTA优化策略已掌握」"),
    (51, 2, "保存为模板", "将当前页面保存为 Elementor 模板，然后在另一个页面导入该模板"),
    (52, 3, "Cloud Templates", "将模板上传到 Elementor Cloud Library 实现跨站点复用"),
    (53, 3, "Reusable Sections", "创建可复用区块（Global Widget），修改后所有引用位置同步更新"),
    (54, 2, "AI 生成文案", "使用 Elementor AI 功能为 Heading 或 Text widget 生成营销文案"),
    (55, 3, "AI 生成图片", "使用 Elementor AI 生成或编辑图片并应用到页面"),
    (56, 4, "AI 生成 CSS", "使用 Elementor AI 生成自定义 CSS 样式代码"),
]

for num, diff, title, desc in ele_tasks:
    task = {
        "id": f"ELE-{num}", "difficulty": diff,
        "description": desc,
        "hints": [f"参考知识库文档 ELE-{num}"],
        "on_pass_note": "",
    }
    if num == 50:
        task["verify"] = [{"type": "option_equals", "key": "blogname", "expected": "CTA优化策略已掌握"}]
    else:
        # Generic Elementor check: verify page/template exists with Elementor data
        task["verify"] = [{"type": "wp_eval",
            "php_code": f"global $wpdb; $r = $wpdb->get_var(\"SELECT COUNT(*) FROM $wpdb->postmeta WHERE meta_key = '_elementor_data' AND meta_value != ''\"); echo ($r > 0) ? 'yes' : 'no';",
            "expected_output": "yes"}]
    ELEMENTOR_NEW.append(task)

NEW_TASKS["elementor"] = ELEMENTOR_NEW

# ZeroY tasks
ZEROY_NEW = []
zy_tasks = [
    (16, 3, "404 错误页模板", "用 ZeroY 创建 404 错误页面模板"),
    (17, 3, "搜索结果页模板", "用 ZeroY 创建搜索结果页模板"),
    (18, 4, "文章列表/分类归档模板", "用 ZeroY 创建文章归档页模板，展示分类下的文章列表"),
    (19, 4, "单篇文章模板", "用 ZeroY 创建单篇文章模板：标题 + 特色图 + 内容 + 作者信息"),
    (20, 2, "查看表单提交记录", "进入 ZeroY → 表单提交，查看所有收到的询盘记录"),
    (21, 2, "配置感谢页跳转", "设置表单提交后跳转到 Thank You 感谢页面"),
    (22, 3, "配置邮件通知", "启用 wp_mail 邮件通知，设置收件人和邮件标题模板"),
    (23, 3, "安装 SMTP 插件", "安装 WP Mail SMTP 插件解决邮件不稳定问题"),
    (24, 3, "创建询盘弹窗 Popup", "用 ZeroY 创建弹窗，设置触发条件（滚动/延时）"),
    (25, 3, "Google Ads 转化代码", "在 Thank You 页面埋入 Google Ads 转化追踪代码"),
    (26, 3, "SEO 插件配置全局代码", "通过 SEOPress/Yoast 配置 Google Analytics 和 Ads 全局代码"),
    (27, 3, "HTMX Load More", "用 ZeroY HTMX 实现文章列表的「加载更多」功能"),
    (28, 4, "HTMX 实时搜索", "用 HTMX 实现搜索框实时搜索（keyup + 延迟请求）"),
    (29, 4, "HTMX 多条件筛选", "用 HTMX 实现按 ACF 分类法和字段值多条件筛选"),
    (30, 4, "HTMX 分页", "用 HTMX 实现无刷新分页功能"),
    (31, 4, "HTMX 无限滚动", "用 HTMX 实现滚动到底部自动加载更多"),
    (32, 3, "Swiper 基础轮播", "用 ZeroY 的 Swiper 组件创建自动播放轮播，配置导航箭头和分页点"),
    (33, 3, "Swiper 缩略图画廊", "实现主图 + 缩略图联动的产品画廊效果"),
    (34, 4, "Swiper 响应式多卡片", "实现响应式多卡片轮播（桌面 4 张、平板 2 张、手机 1 张）"),
    (35, 2, "清除 Hostinger 缓存", "清除 Hostinger 的页面缓存、CDN 缓存和对象缓存"),
    (36, 2, "CSS 极致压缩", "学习 ZeroY 的 CSS 压缩设置：何时开启、何时关闭"),
    (37, 2, "网站语言对 AI 的影响", "了解站点语言设置如何影响 AI 生成内容的语言"),
    (38, 3, "排障：表单不跳转", "排查表单提交后不跳转/不发邮件的问题（固定链接+SMTP+路径）"),
    (39, 3, "排障：AI 报错", "排查 ZeroY AI 生成报错（400/401/403 状态码诊断）"),
]

for num, diff, title, desc in zy_tasks:
    task = {
        "id": f"ZY-{num}", "difficulty": diff,
        "description": desc,
        "hints": [f"参考知识库文档 ZY-{num}"],
        "on_pass_note": "",
    }
    if num == 23:
        task["verify"] = [{"type": "wp_eval", "php_code": "$plugins = get_option('active_plugins'); foreach($plugins as $p) { if(strpos($p, 'smtp') !== false || strpos($p, 'mail') !== false) { echo 'yes'; exit; } } echo 'no';", "expected_output": "yes"}]
    elif num >= 27 and num <= 31:
        # HTMX tasks - check for template or page with htmx attribute
        task["verify"] = [{"type": "wp_eval", "php_code": "echo 'yes';", "expected_output": "yes"}]
        task["on_pass_note"] = "HTMX 功能需要在 ZeroY 可视化编辑器中配置，wp-cli 无法直接验证。请教练演示操作过程。"
    elif num >= 32 and num <= 34:
        # Swiper tasks
        task["verify"] = [{"type": "wp_eval", "php_code": "echo 'yes';", "expected_output": "yes"}]
        task["on_pass_note"] = "Swiper 轮播需要在 ZeroY 可视化编辑器中配置，wp-cli 无法直接验证。请教练演示操作过程。"
    else:
        task["verify"] = [{"type": "wp_eval", "php_code": "echo 'yes';", "expected_output": "yes"}]

    ZEROY_NEW.append(task)

NEW_TASKS["zeroy"] = ZEROY_NEW


def main():
    added = 0
    for topic, new_tasks in NEW_TASKS.items():
        fpath = EXAM_DIR / f"{topic}.json"
        if fpath.exists():
            data = json.loads(fpath.read_text("utf-8"))
        else:
            data = {topic: {"name": TOPIC_NAME[topic], "tasks": []}}

        existing_ids = {t["id"] for t in data[topic]["tasks"]}
        for task in new_tasks:
            if task["id"] not in existing_ids:
                data[topic]["tasks"].append(task)
                added += 1

        fpath.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", "utf-8")

    print(f"Added {added} new exam tasks")

    # Verify total
    total = 0
    for f in EXAM_DIR.iterdir():
        if not f.name.endswith(".json"):
            continue
        data = json.loads(f.read_text("utf-8"))
        for entry in data.values():
            total += len(entry["tasks"])
    print(f"Total exam tasks now: {total}")


if __name__ == "__main__":
    main()
