---
name: leptos
description: "Leptos 0.7 CSR/WASM frontend development guide. Component patterns, signals, view! macro syntax, API calls, i18n, and 25 gotchas for building Leptos WASM apps. Use when writing or reviewing Leptos frontend code."
author: $USER
version: 1.0.0
---

# Leptos 0.7 CSR / WASM Reference

When writing or modifying Leptos frontend code, follow these patterns and avoid these pitfalls.

## Setup

- **Leptos 0.7 CSR** (client-side rendering, WASM target)
- **Build check**: `cargo check --manifest-path app/leptos-ui/Cargo.toml --target=wasm32-unknown-unknown`
- **Crate root**: `lib.rs` needs `#![allow(unused_parens)]` — Leptos view macros require parens around closure attribute values but rustc flags them as unnecessary

## Component Pattern

```rust
use leptos::prelude::*;
use crate::i18n::t;
use crate::state::use_app_state;

#[component]
pub fn MyPage() -> impl IntoView {
    let app_state = use_app_state();
    let (loading, set_loading) = signal(true);
    let (error_msg, set_error_msg) = signal(None::<String>);
    let (data, set_data) = signal(Vec::new());

    // Fetch on mount
    spawn_local(async move {
        set_loading.set(true);
        match api::fetch_something().await {
            Ok(d) => set_data.set(d),
            Err(e) => set_error_msg.set(Some(e)),
        }
        set_loading.set(false);
    });

    view! {
        <div class="my-page">
            <h2>{t("my-page-title")}</h2>
        </div>
    }
}
```

## Signals (State)

```rust
let (value, set_value) = signal(initial);  // ReadSignal + WriteSignal
value.get()           // read (creates reactive dependency — triggers re-render)
value.get_untracked() // read WITHOUT tracking (no re-render)
set_value.set(new)    // write (returns (), not the new value — no chaining)
```

## Global State

```rust
// Provide once in App
provide_context(my_state);

// Use in any component — PANICS if context not provided
let state = use_app_state();  // calls expect_context::<AppState>()

// Safe alternative (returns Option):
let state = use_context::<AppState>();
```

## View Macro Patterns

### Class binding
```rust
// Static
<div class="my-class">

// Dynamic (reactive) — MUST use parens, Leptos macro parser requires them
<div class=(move || if active.get() { "tab active" } else { "tab" })>

// Toggle class
<button class="btn" class:active=move || is_active.get()>
```

### Event handlers
```rust
<button on:click=move |_| set_count.set(count.get() + 1)>
<input on:input=move |ev| set_val.set(event_target_value(&ev)) />
```

### Conditionals
```rust
{move || error.get().map(|msg| view! { <div class="error">{msg}</div> })}
{move || show.get().then(|| view! { <Modal /> })}
```

### Lists
```rust
{items.iter().map(|item| {
    let name = item.name.clone();  // clone before move
    view! { <div>{name}</div> }
}).collect::<Vec<_>>()}
```

### Match arms (tab routing)
```rust
// Each arm MUST call .into_any() to erase concrete types
match current_tab.get() {
    0 => view! { <DashboardPage /> }.into_any(),
    1 => view! { <BeadsPage /> }.into_any(),
    _ => view! { <DashboardPage /> }.into_any(),
}
```

### Text
```rust
<span>"Static text"</span>
<span>{t("i18n-key")}</span>
<span>{move || format!("Count: {}", count.get())}</span>
```

## API Calls

All HTTP in `api.rs` via `web_sys::Request` + `JsFuture` (NOT reqwest — it doesn't work in WASM).

```rust
pub async fn fetch_beads() -> Result<Vec<ApiBead>, String> {
    fetch_json(&format!("{API_BASE}/api/beads")).await
}
```

Usage: always via `spawn_local` (NOT `tokio::spawn`).

## i18n

- Files: `src/locales/en.ftl`, `src/locales/fr.ftl`
- Function: `t("key-name")` — reads current locale from context
- With args: `t_args("key", &args)`
- Always `use crate::i18n::t;`
- FluentBundle is not Send+Sync — stored via `StoredValue<I18n, LocalStorage>`

## Navigation

Tab-based via `current_tab: ReadSignal<usize>` signal — not URL routing.

## Gotchas & Pitfalls

### Macro & Compiler

1. **Parens required in view! class attributes**: `class=(move || ...)` NEEDS the outer parens — the Leptos macro parser requires them. rustc flags these as "unnecessary parentheses" but removing them causes `failed to parse expression` errors. Suppress with `#![allow(unused_parens)]` at crate root.

2. **`.into_any()` on match arms**: When matching on different views, each arm must call `.into_any()` to erase the concrete type. Forgetting this gives opaque type mismatch errors.

3. **`collect::<Vec<_>>()`**: Iterator-based lists in view! must be collected. Forgetting this gives "iterator is not an IntoView" errors.

### Closures & Ownership

4. **Clone before `move ||`**: Closures in view! capture by move. Clone strings/vecs BEFORE the closure:
   ```rust
   let label = item.label.clone();
   let id = item.id;
   view! { <span on:click=move |_| select(id)>{label}</span> }
   ```

5. **Unused signal half**: If you only use the setter, prefix the reader: `let (_read, set_read) = signal(...)`. Otherwise rustc warns.

6. **`set_value.set()` returns `()`**: Signal setters return unit — you cannot chain operations like `.set(x).set(y)`.

### Reactivity

7. **`get()` creates tracking**: Every `.get()` in a `move ||` closure creates a reactive dependency. The closure re-runs when ANY tracked signal changes. Use `.get_untracked()` when you don't want re-renders.

8. **No built-in memoization**: Leptos 0.7 signals don't memoize. If you need it, use `Memo::new(move |_| ...)` explicitly.

9. **Reactive lists clone everything**: `{move || signal.get().into_iter().map(...)}` clones the entire Vec on every change. For large lists, consider `<For>` component or manual diffing.

### WASM-Specific

10. **No `reqwest`**: Use `web_sys::Request` + `wasm_bindgen_futures::JsFuture` for HTTP.

11. **No `tokio::spawn`**: WASM is single-threaded. Use `leptos::task::spawn_local` for all async work.

12. **`serde_wasm_bindgen::from_value()`**: Converting `JsValue` to Rust types requires `serde_wasm_bindgen`, NOT regular `serde_json`. Easy to confuse.

13. **WebSocket URL from HTTP**: Replace protocol manually: `.replace("http://", "ws://").replace("https://", "wss://")`. No built-in helper.

14. **`Closure::wrap(...).forget()`**: WebSocket/event handlers must use `Closure::wrap(Box::new(move |...| {...}))` then `.forget()` to prevent Rust from dropping the closure while JS still references it. This intentionally leaks memory — it's the correct pattern for long-lived handlers.

15. **`gloo_timers` for delays**: Use `gloo_timers::future::TimeoutFuture::new(ms).await` inside `spawn_local` instead of raw `setTimeout` via web_sys.

### State Management

16. **`StoredValue` for non-Send types**: `FluentBundle` has `RefCell` (not Send+Sync). Use `StoredValue<T, LocalStorage>` and access via `.with_value(|v| {...})`. Safe because WASM is single-threaded.

17. **`Rc<RefCell<>>` for shared mutable non-reactive state**: WebSocket handles and other JS interop objects live in `Rc<RefCell<Option<T>>>` because they can't be Leptos signals.

18. **`expect_context()` panics**: If the context provider isn't an ancestor, `expect_context::<T>()` panics at runtime with an unhelpful error. Use `use_context::<T>()` (returns `Option`) if the context might not exist.

### Effects & Lifecycle

19. **`Effect::new()` runs after mount**: Effects run AFTER the component renders, not during. Critical setup (WebSocket connections, timers) happens in Effect bodies, not at component top-level.

20. **No automatic cleanup**: Leptos 0.7 doesn't auto-cleanup WebSocket connections or intervals. Use `on_cleanup(|| {...})` if you need teardown logic, or the connection leaks.

### DOM & Events

21. **`event_target_value(&ev)`**: Built-in helper for text input values. No built-in equivalent for checkboxes — write your own `event_target_checked()`:
    ```rust
    fn event_target_checked(ev: &web_sys::Event) -> bool {
        ev.target().unwrap().dyn_into::<web_sys::HtmlInputElement>().unwrap().checked()
    }
    ```

22. **`NodeRef` requires casting**: `NodeRef` gives you a generic `HtmlElement`. For specific DOM operations (`.set_scroll_top()`, `.focus()`), cast via `let el: &web_sys::HtmlElement = &*node_ref.get().unwrap();`

23. **Drag-and-drop is manual**: No built-in DnD. Use dual signals — one for the dragged item ID, one for the drop target — with `on:dragstart`, `on:dragover`, `on:drop` handlers.

### Data Flow

24. **Optimistic UI has no rollback**: When updating local state then calling the API, there's no automatic rollback on failure. If you update signals optimistically, handle the error case:
    ```rust
    let prev = data.get_untracked();
    set_data.set(new_value);
    spawn_local(async move {
        if let Err(e) = api::save(new_value).await {
            set_data.set(prev);  // rollback
            set_error.set(Some(e));
        }
    });
    ```

25. **Unbounded signal growth**: Signals holding `Vec<T>` (terminal output, logs) grow without limit. Cap manually:
    ```rust
    set_lines.update(|lines| {
        lines.push(new_line);
        if lines.len() > 2000 { lines.drain(..lines.len() - 2000); }
    });
    ```

26. **`on_cleanup` requires `Send + Sync`**: Leptos `on_cleanup(|| {...})` requires the closure to be `Send + Sync`, but `Rc<RefCell<>>` and `Rc<Cell<>>` are not. Since WASM is single-threaded, wrap with `send_wrapper::SendWrapper`:
    ```rust
    let ws_ref_cleanup = send_wrapper::SendWrapper::new(ws_ref.clone());
    on_cleanup(move || {
        if let Some(ws) = ws_ref_cleanup.borrow().as_ref() {
            ws.close().ok();
        }
    });
    ```
    **Gotcha within the gotcha**: `SendWrapper` implements `Deref` AND has its own `.take()` method that unwraps the `SendWrapper` itself. If you have `SendWrapper<Rc<Cell<Option<T>>>>` and call `.take()`, you get the `Rc` back, NOT `Cell::take()`. Use explicit deref: `(*wrapper).take()` to reach the `Cell`.
