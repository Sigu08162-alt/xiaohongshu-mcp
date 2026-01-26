# 小红书搜索调试方案

## 问题
关键词搜索返回空结果

## 快速调试步骤

### 方法 1: 手动在浏览器中调试 (推荐)

1. **打开搜索页面**
   ```
   https://www.xiaohongshu.com/search_result?keyword=咖啡&source=web_explore_feed
   ```

2. **打开 DevTools Console** (按 F12)

3. **复制粘贴以下代码到 Console**:

```javascript
// === 第1步：检查基础结构 ===
console.log('=== 检查基础结构 ===');
console.log('__INITIAL_STATE__ 存在:', !!window.__INITIAL_STATE__);
console.log('search 存在:', !!window.__INITIAL_STATE__?.search);
console.log('feeds 存在:', !!window.__INITIAL_STATE__?.search?.feeds);

// === 第2步：查看 feeds 对象详情 ===
console.log('\n=== feeds 对象详情 ===');
if (window.__INITIAL_STATE__?.search?.feeds) {
    const feeds = window.__INITIAL_STATE__.search.feeds;
    console.log('feeds 类型:', typeof feeds);
    console.log('feeds keys:', Object.keys(feeds));
    console.log('feeds.value:', feeds.value);
    console.log('feeds._value:', feeds._value);
    console.log('feeds 完整对象:', feeds);
}

// === 第3步：尝试当前代码逻辑 ===
console.log('\n=== 尝试当前提取逻辑 ===');
(() => {
    if (window.__INITIAL_STATE__ &&
        window.__INITIAL_STATE__.search &&
        window.__INITIAL_STATE__.search.feeds) {
        const feeds = window.__INITIAL_STATE__.search.feeds;
        const feedsData = feeds.value !== undefined ? feeds.value : feeds._value;
        if (feedsData) {
            console.log('✅ 成功提取数据');
            console.log('数量:', feedsData.length);
            console.log('第一条数据:', feedsData[0]);
            return feedsData;
        } else {
            console.log('❌ feedsData 为空');
            console.log('feeds.value:', feeds.value);
            console.log('feeds._value:', feeds._value);
        }
    } else {
        console.log('❌ 数据路径不完整');
    }
    return null;
})();

// === 第4步：搜索所有可能的数据路径 ===
console.log('\n=== 搜索所有可能的数据位置 ===');
(() => {
    const testPaths = [
        'search.feeds.value',
        'search.feeds._value',
        'search.feeds.data',
        'search.feeds._data',
        'search.noteList',
        'search.items',
        'search.notes',
        'search.result.feeds',
        'search.result.notes'
    ];

    const found = [];
    testPaths.forEach(path => {
        const parts = path.split('.');
        let current = window.__INITIAL_STATE__;

        for (const part of parts) {
            if (!current) break;
            current = current[part];
        }

        if (Array.isArray(current) && current.length > 0) {
            found.push({
                path,
                length: current.length,
                sample: current[0]
            });
            console.log(`✅ 找到数据: ${path} (${current.length} 条)`);
        }
    });

    if (found.length > 0) {
        console.log('\n找到的所有路径:', found);
        return found;
    } else {
        console.log('\n❌ 所有已知路径都未找到数据');
    }
})();

// === 第5步：深度搜索所有数组 ===
console.log('\n=== 深度搜索所有可能的笔记数组 ===');
function findAllArrays(obj, path = '', depth = 0, maxDepth = 4) {
    if (depth >= maxDepth || !obj || typeof obj !== 'object') return [];

    const results = [];
    for (const key in obj) {
        if (!obj.hasOwnProperty(key)) continue;

        const currentPath = path ? `${path}.${key}` : key;
        const value = obj[key];

        if (Array.isArray(value) && value.length > 0) {
            const firstItem = value[0];
            if (firstItem && typeof firstItem === 'object') {
                const hasNoteFields = ('id' in firstItem) ||
                                      ('noteId' in firstItem) ||
                                      ('noteCard' in firstItem) ||
                                      ('feedId' in firstItem);

                if (hasNoteFields) {
                    console.log(`✅ 可能的笔记数组: ${currentPath} (${value.length} 条)`);
                    console.log('   第一项 keys:', Object.keys(firstItem).slice(0, 15).join(', '));
                    results.push({ path: currentPath, data: value });
                }
            }
        }

        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            results.push(...findAllArrays(value, currentPath, depth + 1, maxDepth));
        }
    }
    return results;
}

if (window.__INITIAL_STATE__?.search) {
    const arrays = findAllArrays(window.__INITIAL_STATE__.search);
    if (arrays.length > 0) {
        console.log('\n找到的笔记数组:', arrays.length, '个');
        console.table(arrays.map(a => ({
            路径: a.path,
            数量: a.data.length,
            第一项ID: a.data[0]?.id || a.data[0]?.noteId || 'N/A'
        })));
    } else {
        console.log('\n❌ 未找到任何笔记数组');
    }
}

console.log('\n=== 调试完成 ===');
console.log('如果看到 ✅ 说明找到了数据');
console.log('请将找到的路径告知开发者以更新代码');
```

### 方法 2: 检查网络请求

1. 打开 DevTools **Network** 标签
2. 刷新页面
3. 筛选 **Fetch/XHR**
4. 查找包含 `search` 或 `feed` 的请求
5. 查看响应数据，确认数据格式

### 方法 3: 使用测试命令 (需要先登录)

```bash
# 在项目目录运行
cd /Users/xumingyang/app/xiaohongshu-mcp

# 使用现有的 MCP 工具测试搜索
# 方法1: 通过 MCP 调用
echo '{"method":"tools/call","params":{"name":"search_feeds","arguments":{"keyword":"咖啡"}}}' | ./xiaohongshu-mcp

# 方法2: 查看日志
tail -f logs/*.log
```

## 常见问题和解决方案

### 问题1: feeds.value 和 feeds._value 都是 undefined

**可能原因**:
- Vue 3 的响应式系统改变了数据包装方式
- 小红书更新了数据结构

**解决方案**:
在 Console 中运行:
```javascript
const feeds = window.__INITIAL_STATE__.search.feeds;
console.log('所有 keys:', Object.keys(feeds));
console.log('所有属性:', Object.getOwnPropertyNames(feeds));
console.log('Symbol keys:', Object.getOwnPropertySymbols(feeds));

// 尝试访问所有可见属性
for (let key in feeds) {
    console.log(`feeds.${key}:`, feeds[key]);
}
```

### 问题2: search.feeds 不存在

**可能原因**:
- 数据路径改变
- 页面还在加载

**解决方案**:
```javascript
// 查看 search 下的所有内容
console.log('search 的所有 keys:', Object.keys(window.__INITIAL_STATE__.search));

// 递归查找所有数组
function findArrays(obj, prefix = '') {
    for (let key in obj) {
        const val = obj[key];
        if (Array.isArray(val)) {
            console.log(`数组: ${prefix}${key}, 长度: ${val.length}`);
        } else if (typeof val === 'object' && val !== null) {
            findArrays(val, `${prefix}${key}.`);
        }
    }
}
findArrays(window.__INITIAL_STATE__.search, 'search.');
```

### 问题3: __INITIAL_STATE__ 不存在

**可能原因**:
- 页面还未加载完成
- 小红书改用了异步加载

**解决方案**:
```javascript
// 等待数据加载
async function waitForData() {
    for (let i = 0; i < 60; i++) {
        if (window.__INITIAL_STATE__?.search?.feeds) {
            console.log('数据已加载!');
            return true;
        }
        await new Promise(r => setTimeout(r, 500));
    }
    console.log('等待超时');
    return false;
}

await waitForData();
```

## 预期结果

成功时应该看到类似输出:
```
✅ 成功提取数据
数量: 20
第一条数据: {id: "xxx", noteCard: {...}, ...}
```

失败时请截图所有输出，特别是:
- feeds 对象的完整结构
- 找到的所有数据路径
- Network 标签中的相关请求

## 报告模板

如果遇到问题，请提供以下信息:

```
1. 浏览器版本: ___________
2. 搜索关键词: ___________
3. __INITIAL_STATE__ 是否存在: 是/否
4. search.feeds 是否存在: 是/否
5. 找到的数据路径: ___________
6. Console 输出截图: (附件)
7. Network 请求截图: (附件)
```

## 代码修复位置

如果找到了正确的数据路径,需要修改:
- 文件: `xiaohongshu/search.go`
- 行号: 266-277
- 当前逻辑: `feeds.value || feeds._value`
- 需要更新为: (根据调试结果)
