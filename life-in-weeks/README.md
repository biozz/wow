# Life in Weeks

This folder contains a small \"life in weeks\" visualization that is driven by external data.

To provide your own data:

1. Edit `data.yaml` using the structure from `data.example.yaml`.
2. Convert it to JSON with `yq`:

```bash
yq -o=json data.yaml > data.json
```

You must run the conversion manually **before serving** the app so that `main.js` can load `data.json`.

