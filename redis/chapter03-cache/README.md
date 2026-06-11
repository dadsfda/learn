# 第三章：缓存实战

> Redis 最常见的用途是缓存，但缓存一定要考虑失效和异常情况。

## 目标

- 理解缓存的基本读写流程
- 掌握常见缓存问题：穿透、击穿、雪崩
- 学会设计简单可靠的缓存 key

## 1. 最基础的缓存流程

以商品详情接口为例：

```text
用户请求商品详情
-> 查 Redis
-> 命中：直接返回
-> 未命中：查数据库
-> 写入 Redis
-> 返回给用户
```

命令模拟：

```bash
GET product:1001:detail
SET product:1001:detail "{\"id\":1001,\"name\":\"机械键盘\",\"price\":299}" EX 600
GET product:1001:detail
```

## 2. 缓存更新策略

实际项目里最常用的是：

```text
更新数据库成功后，删除缓存。
下一次查询时，再重新加载缓存。
```

为什么不是直接更新缓存？因为真实业务里商品详情可能来自多张表，直接拼出新缓存容易漏字段或写错。

示例：

```bash
DEL product:1001:detail
```

下一次查询发现缓存不存在，再查数据库并重建缓存。

## 3. 缓存穿透

缓存穿透：用户请求一个数据库里也不存在的数据，例如 `product_id=-1`。Redis 查不到，数据库也查不到，如果一直请求会打到数据库。

简单解决方式：缓存空结果，时间设置短一点。

```bash
SET product:-1:detail "" EX 60
```

业务判断时：

- key 不存在：查数据库
- key 存在但值为空：直接返回“不存在”

## 4. 缓存击穿

缓存击穿：某个热点 key 过期的一瞬间，大量请求同时打到数据库。

常见解决方式：

- 热点 key 设置更长过期时间
- 后台提前刷新热点缓存
- 使用互斥锁，只允许一个请求回源数据库

简单互斥锁示例：

```bash
SET lock:product:1001 1 NX EX 10
```

返回 `OK` 表示拿到锁，当前请求负责查数据库并重建缓存；没拿到锁的请求可以稍等后再查 Redis。

## 5. 缓存雪崩

缓存雪崩：大量 key 在同一时间过期，数据库突然承受大量请求。

简单解决方式：给过期时间加随机值。

```text
商品详情基础过期时间 600 秒
实际过期时间 = 600 + 0 到 120 秒随机值
```

这样不同 key 不会同一秒集中失效。

## 6. Go 伪代码：缓存商品详情

```go
func GetProductDetail(id int64) (string, error) {
    key := fmt.Sprintf("product:%d:detail", id)

    value, err := redis.Get(ctx, key).Result()
    if err == nil {
        return value, nil
    }
    if err != redis.Nil {
        return "", err
    }

    product, err := queryProductFromDB(id)
    if err != nil {
        return "", err
    }

    if product == "" {
        redis.Set(ctx, key, "", time.Minute)
        return "", nil
    }

    ttl := 10*time.Minute + time.Duration(rand.Intn(120))*time.Second
    redis.Set(ctx, key, product, ttl)
    return product, nil
}
```

重点不是代码库，而是流程：

1. Redis 命中就返回。
2. Redis 未命中才查数据库。
3. 数据库没有数据，也短暂缓存空值。
4. 正常数据设置带随机值的过期时间。

## 实际应用：接口响应提速

适合缓存的数据：

- 读多写少
- 允许短时间不一致
- 查询数据库成本较高
- 结果体积不太大

不适合缓存的数据：

- 强一致余额
- 高频变化库存扣减
- 超大对象
- 每个用户都不同且访问频率很低的数据

## 练习

1. 设计 `article:1001:detail` 的缓存 key。
2. 给商品缓存设置 10 分钟过期时间。
3. 模拟一个不存在的商品，缓存空结果 30 秒。
4. 思考：用户修改昵称后，应该更新缓存还是删除缓存？

## 下一章

[第四章：登录会话与验证码](../chapter04-session-token/) — 学习 Redis 保存临时状态。

