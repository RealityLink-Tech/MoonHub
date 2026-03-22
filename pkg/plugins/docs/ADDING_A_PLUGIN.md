# Adding a Built-in Plugin (Checklist)

## General

1. Create `plugin.go` in `pkg/plugins/<channels|providers|tools>/<newname>/`.
2. In **`init()`**, call `plugin.RegisterPlugin(&YourPlugin{})` (import `github.com/RealityLink-Tech/MoonHub/pkg/framework`, use package prefix **`plugin`** in code).
3. Implement **`Metadata()`**, `Init(*plugin.RuntimeContext)`, `Validate(*config.Config)` and all methods of corresponding sub-interface.
4. Add a line `_ "github.com/RealityLink-Tech/MoonHub/pkg/plugins/.../newname"` in **`cmd/moonhub/internal/gateway/helpers.go`**, grouped with similar plugins.
5. Run `go build ./...`, add tests if necessary.

## Channel Plugins

- Implement `plugin.ChannelPlugin`: `ChannelPrefix`, `CreateChannel`, `IsEnabled`.
- `CreateChannel` typically delegates to existing constructors in `pkg/channels/<impl>`.
- Verify `channels.Manager` compatibility with return type and `channels.Channel`, optional `SetMediaStore` / `SetPlaceholderRecorder` / `SetOwner`.

## Provider Plugins

- Implement `ProviderPlugin`, with special attention to:
  - **`SupportsProtocol`** must match protocol prefix in model configuration.
  - **`CreateProviderFromModelConfig`**: return **`providers.ErrSkipProvider`** when not applicable to current config, allowing resolver to try next plugin or fallback to built-in factory.
- Can reuse exported factory functions in `pkg/providers` (e.g., OAuth-related).

## Tool Plugins

- Implement `ToolPlugin`: `CreateTools`, `IsCore` (determines whether to register as **visible** to LLM).
- If tool name duplicates Agent built-in `registerSharedTools` (e.g., `web_search`, `web_fetch`, `message`), ensure Gateway has merged via plugin Registry to avoid duplicate registration; see skip logic in `pkg/agent/loop.go`.

## Documentation

- Update [`PLUGIN_INDEX.md`](./PLUGIN_INDEX.md) and lists in this directory or repository-level `docs/implementation/plugin-status.md` (if project requires keeping them in sync).
