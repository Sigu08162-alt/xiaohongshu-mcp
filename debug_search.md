# 小红书搜索调试指南

## 问题描述
关键词搜索返回空结果

## 调试步骤

### 1. 使用 Chrome DevTools 手动验证

1. **打开搜索页面**
   ```
   https://www.xiaohongshu.com/search_result?keyword=咖啡&source=web_explore_feed
   ```

2. **打开 DevTools Console** (F12)

3. **检查 __INITIAL_STATE__ 对象**
   ```javascript
   // 检查对象是否存在
   console.log('__INITIAL_STATE__ 存在:', !!window.__INITIAL_STATE__);
   console.log('search 存在:', !!window.__INITIAL_STATE__?.search);
   console.log('feeds 存在:', !!window.__INITIAL_STATE__?.search?.feeds);

   // 查看 feeds 结构
   console.log('feeds 对象:', window.__INITIAL_STATE__?.search?.feeds);
   console.log('feeds keys:', Object.keys(window.__INITIAL_STATE__?.search?.feeds || {}));

   // 尝试不同的访问方式
   const feeds = window.__INITIAL_STATE__?.search?.feeds;
   console.log('feeds.value:', feeds?.value);
   console.log('feeds._value:', feeds?._value);
   console.log('feeds.data:', feeds?.data);
   console.log('feeds._data:', feeds?._data);
   ```

4. **提取搜索结果 (复制当前代码逻辑)**
   ```javascript
   (() => {
       if (window.__INITIAL_STATE__ &&
           window.__INITIAL_STATE__.search &&
           window.__INITIAL_STATE__.search.feeds) {
           const feeds = window.__INITIAL_STATE__.search.feeds;
           const feedsData = feeds.value !== undefined ? feeds.value : feeds._value;
           if (feedsData) {
               console.log('找到 feedsData, 数量:', feedsData.length);
               console.log('第一条:', feedsData[0]);
               return JSON.stringify(feedsData);
           } else {
               console.log('feedsData 为空');
               console.log('feeds 对象完整结构:', feeds);
           }
       } else {
           console.log('路径不存在');
       }
       return "";
   })();
   ```

### 2. 检查数据结构变化

小红书可能更新了数据结构。检查以下可能性：

1. **Vue 3 响应式对象变化**
   - `value` -> `_value`
   - 可能使用了新的响应式包装

2. **数据路径变化**
   - `search.feeds` -> `search.notes`
   - `search.feeds` -> `search.items`
   - `search.feeds` -> `search.noteList`

3. **完整枚举所有可能的路径**
   ```javascript
   // 在 Console 中运行
   function findFeeds(obj, path = '') {
       for (let key in obj) {
           if (obj.hasOwnProperty(key)) {
               const currentPath = path ? `${path}.${key}` : key;
               const value = obj[key];

               if (Array.isArray(value) && value.length > 0) {
                   console.log(`数组路径: ${currentPath}, 长度: ${value.length}`);
                   if (value[0].id || value[0].noteId || value[0].feedId) {
                       console.log('可能是笔记数组!', currentPath);
                       console.log('第一项:', value[0]);
                   }
               }

               if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                   findFeeds(value, currentPath);
               }
           }
       }
   }

   findFeeds(window.__INITIAL_STATE__);
   ```

### 3. 网络请求检查

1. **查看 Network 标签**
   - 过滤: `XHR`, `Fetch`
   - 查找包含 `search` 或 `feed` 的请求
   - 检查响应数据格式

2. **查看搜索 API**
   ```
   可能的 API 端点:
   - /api/sns/web/v1/search/notes
   - /api/sns/web/v2/search/notes
   - /fe_api/burdock/web/v1/search/notes
   ```

### 4. 检查页面加载时机

```javascript
// 等待数据加载的改进版本
async function waitForSearchData(maxWaitMs = 30000) {
    const checkInterval = 500;
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitMs) {
        // 尝试多种可能的路径
        const paths = [
            'search.feeds.value',
            'search.feeds._value',
            'search.feeds.data',
            'search.feeds._data',
            'search.noteList',
            'search.items',
            'search.notes.value',
            'search.notes._value'
        ];

        for (const path of paths) {
            const parts = path.split('.');
            let current = window.__INITIAL_STATE__;

            for (const part of parts) {
                if (!current) break;
                current = current[part];
            }

            if (Array.isArray(current) && current.length > 0) {
                console.log(`找到数据在路径: ${path}, 数量: ${current.length}`);
                return { path, data: current };
            }
        }

        await new Promise(resolve => setTimeout(resolve, checkInterval));
    }

    console.log('超时，未找到搜索数据');
    return null;
}

// 使用
waitForSearchData().then(result => {
    if (result) {
        console.log('成功:', result);
    }
});
```

## 预期输出

### 正常情况
```javascript
{
    path: "search.feeds.value",
    data: [
        {
            id: "xxx",
            noteId: "xxx",
            title: "标题",
            user: {...},
            // ...其他字段
        },
        // ...更多笔记
    ]
}
```

### 异常情况需要记录
1. `window.__INITIAL_STATE__` 不存在
2. 数据在其他路径
3. 需要等待特定的异步请求完成

## 修复建议

根据调试结果，可能需要：

1. **更新数据提取路径** (search.go:266-277)
2. **增加更多数据路径尝试**
3. **等待特定的网络请求完成**
4. **检查页面 URL 参数是否正确**

## 调试命令

```bash
# 运行测试搜索
go run cmd/test_search/main.go
```
