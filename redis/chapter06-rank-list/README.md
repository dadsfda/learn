# 第六章：排行榜

> Redis 的 ZSet 非常适合实现排行榜、热度榜和 Top N 查询。

## 目标

- 使用 ZSet 保存分数和成员
- 查询排行榜前 N 名
- 查询用户自己的排名和分数

## 1. 创建积分榜

```bash
ZADD game:rank 9800 user:1001
ZADD game:rank 7600 user:1002
ZADD game:rank 12000 user:1003
ZADD game:rank 8800 user:1004
```

查询从高到低前 3 名：

```bash
ZREVRANGE game:rank 0 2 WITHSCORES
```

查询从低到高：

```bash
ZRANGE game:rank 0 2 WITHSCORES
```

## 2. 更新用户分数

直接设置新分数：

```bash
ZADD game:rank 13000 user:1001
```

在原分数基础上增加：

```bash
ZINCRBY game:rank 500 user:1001
```

## 3. 查询自己的排名

从高到低的排名：

```bash
ZREVRANK game:rank user:1001
```

Redis 返回的排名从 0 开始。显示给用户时通常要加 1。

查询自己的分数：

```bash
ZSCORE game:rank user:1001
```

## 4. 查询某个分数范围

查询分数在 8000 到 12000 之间的用户：

```bash
ZRANGEBYSCORE game:rank 8000 12000 WITHSCORES
```

## 5. 实际应用：文章热度榜

可以把文章 ID 作为成员，热度作为分数：

```bash
ZADD article:hot 10 article:1001
ZINCRBY article:hot 1 article:1001
ZINCRBY article:hot 5 article:1002
ZREVRANGE article:hot 0 9 WITHSCORES
```

热度分数可以来自：

- 浏览 +1
- 点赞 +5
- 评论 +10
- 收藏 +8

这样就可以快速拿到当前最热的文章。

## 6. 排行榜 key 设计

不同周期应该用不同 key：

```text
rank:game:all
rank:game:daily:2026-06-06
rank:game:weekly:2026-W23
rank:article:hot:daily:2026-06-06
```

每日榜可以设置过期时间：

```bash
EXPIRE rank:game:daily:2026-06-06 2592000
```

保留 30 天即可，避免无限占用内存。

## 7. Go 伪代码：查询 Top 10

```go
func TopPlayers() ([]redis.Z, error) {
    return redis.ZRevRangeWithScores(ctx, "game:rank", 0, 9).Result()
}

func PlayerRank(userID int64) (int64, error) {
    key := "game:rank"
    member := fmt.Sprintf("user:%d", userID)

    rank, err := redis.ZRevRank(ctx, key, member).Result()
    if err != nil {
        return 0, err
    }

    return rank + 1, nil
}
```

## 练习

1. 创建一个 `course:rank`，保存 5 个学生分数。
2. 查询前 3 名。
3. 给某个学生增加 20 分。
4. 查询该学生的排名和分数。

## 下一章

[第七章：队列与 Stream](../chapter07-queue-stream/) — 学习异步任务和消息消费。

