# Simple logger lib for zap

## Example
```go

    package main

    import (
        "context"

        "github.com/AlexMickh/logger/pkg/logger"
	    "go.uber.org/zap"
    )

    func main() {
        ctx := context.Background()
        ctx, err := logger.New(ctx, "dev")
        // error handling
        logger.GetFromCtx(ctx).Info(ctx, "hello world")
        ctx = logger.GetFromCtx(ctx).With(ctx, zap.String("hello", "mom"))
    }
```

In `New` function use `dev` for local text logger or `prod` for production json logger