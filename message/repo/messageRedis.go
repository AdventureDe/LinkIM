package repo

// import (
//     "context"
//     "time"

//     "github.com/go-redis/redis/v8"
// )

// type MessageRedis interface {
//     Get(ctx context.Context, key string) *redis.StringCmd
//     Set(ctx context.Context, key string, val interface{}, exp time.Duration) *redis.StatusCmd
//     Del(ctx context.Context, keys ...string) *redis.IntCmd
// }

// type messageRedis struct {
//     client *redis.Client
// }

// func NewMessageRedis(r *redis.Client) MessageRedis {
//     return &messageRedis{
//         client: r,
//     }
// }

// func (m *messageRedis) Get(ctx context.Context, key string) *redis.StringCmd {
//     return m.client.Get(ctx, key)
// }

// func (m *messageRedis) Set(ctx context.Context, key string, val interface{}, exp time.Duration) *redis.StatusCmd {
//     return m.client.Set(ctx, key, val, exp)
// }

// func (m *messageRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
//     return m.client.Del(ctx, keys...)
// }
