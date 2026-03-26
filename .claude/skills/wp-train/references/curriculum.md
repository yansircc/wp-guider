# WordPress 培训课纲

八层递进，从基础操作到深度排障。

## Layer 1: 初识 WordPress

### 1.1 登录与仪表盘
- 访问 /wp-admin/，登录流程
- 仪表盘首页概览（欢迎面板、快速草稿、站点健康）
- 左侧菜单结构：文章、媒体、页面、外观、插件、用户、设置
- 管理栏（Admin Bar）的作用
- 验证: `locwp wp <site> -- user get admin --format=json`

### 1.2 站点基础设置
- 常规设置：站点标题、副标题、时区、日期格式
- 阅读设置：首页显示（最新文章 vs 静态页面）
- 固定链接设置：选择「文章名」结构
- 验证: `locwp wp <site> -- option get blogname`, `option get permalink_structure`

### 1.3 用户与个人资料
- 修改管理员昵称和显示名称
- 头像（Gravatar 原理）
- 用户角色概念（管理员/编辑/作者/投稿者/订阅者）
- 验证: `locwp wp <site> -- user list --format=json`, `option get default_role`

## Layer 2: 内容管理

### 2.1 页面管理
- 创建页面：标题、内容、别名（slug）
- 页面状态：草稿/待审/已发布/私密
- 页面层级（父页面）与排序
- 设置静态首页和文章页
- 验证: `locwp wp <site> -- post list --post_type=page --format=json`

### 2.2 文章与分类体系
- 创建文章：标题、内容、摘要
- 分类（Category）：层级结构，一篇文章可属于多个分类
- 标签（Tag）：扁平结构，灵活标记
- 特色图片（Featured Image）
- 文章格式（Post Format）
- 验证: `locwp wp <site> -- post list --format=json`, `term list category --format=json`

### 2.3 媒体管理
- 上传图片/文件（媒体库）
- 图片尺寸：缩略图/中等/大/原始（Settings → Media）
- 在内容中插入媒体
- 附件页面和文件 URL 结构（/wp-content/uploads/YYYY/MM/）
- 验证: `locwp wp <site> -- media list --format=json`, 检查 uploads 目录结构

### 2.4 区块编辑器基础
- 段落、标题、图片、列表等常用区块
- 区块操作：移动、复制、删除、转换类型
- 可复用区块（Reusable Blocks / Patterns）
- 区块组（Group）和列（Columns）布局
- 验证: `locwp wp <site> -- post get <id> --field=post_content`（检查区块标记）

### 2.5 菜单与导航
- 创建导航菜单（Appearance → Menus）
- 添加菜单项：页面、分类、自定义链接
- 菜单位置（Menu Locations）
- 子菜单（嵌套层级）
- 验证: `locwp wp <site> -- menu list --format=json`, `menu item list <id> --format=json`

### 2.6 外观基础
- 安装主题（从官方仓库 / 上传 zip）
- 激活与预览主题
- 自定义器（Customizer）：站点标识、颜色、菜单、小工具
- 小工具（Widgets）：侧边栏和页脚区域
- 验证: `locwp wp <site> -- theme list --format=json`, `widget list <sidebar-id> --format=json`

### 2.7 插件管理
- 安装插件（搜索 / 上传）
- 激活、停用、删除
- 插件更新与兼容性
- 常见必备插件类型：SEO、安全、缓存、表单、备份
- 验证: `locwp wp <site> -- plugin list --format=json`

## Layer 3: WordPress 文件系统与引导

### 3.1 文件结构
- wp-content/ 的角色（主题、插件、上传）
- wp-config.php 关键配置项（DB、DEBUG、SALT、ABSPATH）
- wp-includes/ vs wp-admin/（前台 vs 后台）
- wp-content/uploads/ 的年月目录结构
- 验证: `locwp wp <site> -- eval "echo ABSPATH;"`, 检查文件是否存在

### 3.2 请求生命周期
- 浏览器 → DNS → Nginx → .htaccess/index.php → wp-blog-header.php
- wp-load.php → wp-config.php → wp-settings.php
- 关键 action 序列: muplugins_loaded → plugins_loaded → after_setup_theme → init → wp → template_redirect → wp_head → the_content → wp_footer
- 验证: `locwp wp <site> -- eval "echo did_action('init');"`, 读 .htaccess 内容

### 3.3 Hook 系统
- Action（做事）vs Filter（改数据）
- add_action / add_filter / do_action / apply_filters
- 优先级（priority）和参数数量
- remove_action / remove_filter
- 验证: 在 functions.php 中添加 hook，用 `wp eval` 验证效果

## Layer 4: 数据层

### 4.1 核心表结构
- wp_posts: 万物皆 post（page, attachment, revision, nav_menu_item, custom）
- wp_postmeta: KV 扩展（_edit_last, _thumbnail_id, 自定义字段）
- wp_options: 全局配置（siteurl, blogname, active_plugins, sidebars_widgets）
- wp_terms / wp_term_taxonomy / wp_term_relationships: 分类体系
- wp_users / wp_usermeta: 用户与角色
- wp_comments / wp_commentmeta
- 验证: `locwp wp <site> -- db query "SHOW TABLES;" --format=csv`

### 4.2 WP_Query
- 主查询 vs 自定义查询
- query_vars 与条件标签（is_single, is_page, is_archive）
- pre_get_posts 修改主查询
- meta_query / tax_query
- 分页机制（posts_per_page, offset, paged）
- 验证: `locwp wp <site> -- post list` 各种 filter 参数

### 4.3 Options API 与 Transients
- get_option / update_option / delete_option
- autoload 机制（yes/no 的性能影响）
- Transient API（set_transient / get_transient / 过期机制）
- Object Cache（内存缓存 vs 持久化缓存）
- 验证: `locwp wp <site> -- option list --autoload=yes --format=csv | wc -l`

## Layer 5: 主题引擎

### 5.1 模板层级（Template Hierarchy）
- 决策树: single-{post_type}-{slug}.php → single-{post_type}.php → single.php → singular.php → index.php
- page 模板: page-{slug}.php → page-{id}.php → page.php
- archive 模板: archive-{post_type}.php → archive.php
- 特殊: front-page.php, home.php, 404.php, search.php
- Template Parts: get_template_part(), header.php, footer.php, sidebar.php
- 验证: 创建特定模板文件后检查页面使用了哪个模板

### 5.2 子主题
- 为什么不能直接改父主题（更新会覆盖）
- style.css 的 Template 声明
- 函数覆盖 vs hook 覆盖
- 模板文件覆盖优先级
- 验证: `locwp wp <site> -- theme list`, 检查 style.css 内容

### 5.3 Theme.json 与 Block Theme
- theme.json 配置: settings（颜色、字体、间距）、styles（全局样式）
- Block template 与 template parts
- FSE（Full Site Editing）原理
- 经典主题 vs Block 主题的区别
- 验证: 读取 theme.json 内容，检查配置是否生效

### 5.4 functions.php
- 本质: 主题激活时自动加载的 PHP 文件
- 注册: add_theme_support(), register_nav_menus(), register_sidebar()
- 入队: wp_enqueue_style(), wp_enqueue_script()
- 与插件的区别（主题切换后失效）
- 验证: 在 functions.php 添加功能，wp eval 验证

## Layer 6: 插件与扩展

### 6.1 插件架构
- 插件头注释（Plugin Name, Version, etc）
- 激活/停用/卸载 hook（register_activation_hook, etc）
- 插件加载顺序与优先级
- Must-Use 插件（mu-plugins/，无需激活，优先加载）
- 验证: `locwp wp <site> -- plugin list --format=json`

### 6.2 自定义 Post Type 与 Taxonomy
- register_post_type() 参数详解
- register_taxonomy() 与关联
- Meta Box 注册（add_meta_box, save_post hook）
- 验证: `locwp wp <site> -- post-type list --format=json`

### 6.3 三代扩展机制
- Shortcode: add_shortcode()（最老，仍广泛使用）
- Widget: WP_Widget 类（经典侧边栏，逐渐淘汰）
- Block: block.json + register_block_type()（现代标准）
- 验证: `locwp wp <site> -- widget list <sidebar-id>`

### 6.4 REST API
- 默认端点: /wp-json/wp/v2/posts, /pages, /users
- 自定义端点: register_rest_route()
- 认证: Nonce / Application Password / OAuth
- 权限回调: permission_callback
- 验证: `locwp wp <site> -- eval "echo rest_url();"`, curl 测试端点

## Layer 7: HTTP 与服务器层

### 7.1 .htaccess 与重写规则
- WordPress 默认 .htaccess 规则（mod_rewrite）
- WP_Rewrite 类（add_rewrite_rule, flush_rewrite_rules）
- 固定链接结构（%postname%, %category%, 自定义）
- Nginx 等效配置（try_files）
- 验证: 读取 .htaccess, `locwp wp <site> -- rewrite list --format=json`

### 7.2 wp-cron
- WP-Cron 原理（页面访问触发，非系统级）
- wp_schedule_event / wp_schedule_single_event
- 为什么 WP-Cron 不准（低流量站点问题）
- DISABLE_WP_CRON + 系统 cron 替代方案
- 验证: `locwp wp <site> -- cron event list --format=json`

### 7.3 安全基础
- Nonce 机制（wp_create_nonce, wp_verify_nonce）
- 数据净化三件套: sanitize_*(), esc_*(), wp_kses()
- 文件权限（755/644 规范）
- wp-config.php 安全配置（SALT、DISALLOW_FILE_EDIT）
- 验证: `locwp wp <site> -- config get DISALLOW_FILE_EDIT`

## Layer 8: 排障

### 8.1 调试工具
- WP_DEBUG / WP_DEBUG_LOG / WP_DEBUG_DISPLAY / SAVEQUERIES
- debug.log 位置与阅读
- Query Monitor 插件
- 验证: `locwp wp <site> -- config set WP_DEBUG true --raw`

### 8.2 常见问题诊断
- 白屏（WSOD）: 内存限制、语法错误、插件冲突
- 404 问题: 重写规则失效 → flush rewrite
- 插件冲突隔离: 逐个停用 / mu-plugins 诊断脚本
- 数据库问题: wp db repair, wp db check
- 验证: 模拟故障场景，教练定位并修复

### 8.3 性能诊断
- autoload bloat（wp_options 表 autoload=yes 过多）
- 慢查询（SAVEQUERIES + 分析）
- 外部 HTTP 请求阻塞（wp_remote_get 超时）
- 对象缓存缺失
- 验证: `locwp wp <site> -- db query "SELECT SUM(LENGTH(option_value)) FROM wp_options WHERE autoload='yes';" --format=csv`

## 知识点编码

格式: `L{层}.{节}.{序号}`

示例:
- `L1.1.1` = Layer 1, 登录与仪表盘, 第 1 个知识点
- `L2.5.2` = Layer 2, 菜单与导航, 第 2 个知识点
- `L8.2.1` = Layer 8, 常见问题诊断, 第 1 个知识点
