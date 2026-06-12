made by claude
# Widget 模組系統

## 概述

首頁看板是一個可自由排列的格線畫布，使用者可在上面新增、移動、縮放並刪除 Widget 卡片。每個 Widget 類型由一個繼承 `WidgetModule` 的類別定義，透過 `WidgetRegistry` 在啟動時一次性注冊，之後即可在看板中使用。

| 層級 | 檔案 | 說明 |
|------|------|------|
| 狀態 | `frontend/lib/features/board/bloc/board_bloc.dart` | BLoC，管理看板上的所有 Widget |
| UI | `frontend/lib/features/board/view/board_grid.dart` | 看板畫布，含拖移 / 縮放 / 新增 / 刪除 |
| 選擇器 | `frontend/lib/features/board/widgets/widget_picker_sheet.dart` | 新增 Widget 的 bottom sheet |

---

## 核心概念

### WidgetModule（介面）

```dart
abstract class WidgetModule {
  String get typeId;           // 唯一識別碼，snake_case，例如 'clock'
  String get label;            // 顯示名稱
  String get icon;             // Emoji 或 Unicode 圖示
  List<Color> get cardColors;  // 漸層背景顏色（至少 2 色）
  List<SizePreset> get supportedPresets;
  SizePreset get defaultPreset;

  Map<String, dynamic> defaultData();        // 建立時的初始資料
  Map<String, dynamic> serialise(data);      // 寫入 Hive 前轉換（預設 identity）
  Map<String, dynamic> deserialise(raw);     // 從 Hive 讀回後轉換（預設 identity）

  Widget buildContent({
    required String instanceId,
    required Map<String, dynamic> data,
    required void Function(Map<String, dynamic>) onUpdate,
  });

  Widget? buildSettings({...}) => null;  // 選填：設定面板
  void onRemove(String instanceId) {}    // 選填：移除時清理
}
```

### WidgetRegistry（單例）

```dart
WidgetRegistry.instance.registerAll([MyModule(), ...]);  // 啟動時調用一次
WidgetRegistry.instance.find('my_module');               // 安全查找（回傳 null）
WidgetRegistry.instance.get('my_module');                // 強制查找（找不到會拋錯）
WidgetRegistry.instance.all;                             // 所有已注冊的模組
```

### BoardWidgetModel（資料模型）

```dart
class BoardWidgetModel {
  final String id;      // UUID v4 實例 ID
  final String typeId;  // 對應 WidgetModule.typeId
  final int gridRow, gridCol;    // 格線位置（0-based）
  final int rowSpan, colSpan;    // 格線跨度
  final Map<String, dynamic> data; // 模組自訂資料
}
```

---

## 格線系統

看板為 **8 欄 × 10 列** 的固定格線。

| 常數 | 值 | 說明 |
|------|----|------|
| `kCellSize` | `80.0 px` | 每格的邊長 |
| `kCellGap` | `8.0 px` | 格間距 |
| `kGridColumns` | `8` | 欄數 |
| `kGridRows` | `10` | 列數 |

**像素換算公式：**

```
pixelWidth  = colSpan × (kCellSize + kCellGap) − kCellGap
pixelHeight = rowSpan × (kCellSize + kCellGap) − kCellGap
pixelLeft   = gridCol × (kCellSize + kCellGap)
pixelTop    = gridRow × (kCellSize + kCellGap)
```

---

## 六種預設尺寸

| 常數 | rows × cols | 像素（W × H） | 適用場景 |
|------|------------|----------------|----------|
| `SizePreset.small`  | 1 × 2 | 168 × 80   | 小標籤、倒數、時鐘（橫式）|
| `SizePreset.medium` | 2 × 2 | 168 × 168  | 標準方形卡片 |
| `SizePreset.large`  | 2 × 4 | 344 × 168  | 寬幅資訊卡 |
| `SizePreset.wide`   | 1 × 4 | 344 × 80   | 橫幅通知列 |
| `SizePreset.tall`   | 4 × 2 | 168 × 344  | 直式列表 |
| `SizePreset.huge`   | 4 × 4 | 344 × 344  | 全尺寸地圖 / 大面板 |

> **提示**：在 `buildContent` 內使用 `LayoutBuilder` 可取得實際像素尺寸，
> 並據此切換不同的排版（例如小尺寸只顯示圖示、大尺寸顯示完整資訊）。

---

## 新增 Widget 模組（步驟）

### 1. 建立模組檔案

```
frontend/lib/features/board/widgets/<name>_module.dart
```

實作範例：

```dart
import 'package:flutter/material.dart';
import '../bloc/board_bloc.dart';

class MyModule extends WidgetModule {
  const MyModule();

  @override String get typeId => 'my_module';
  @override String get label  => '我的 Widget';
  @override String get icon   => '⭐';

  @override
  List<Color> get cardColors => const [Color(0xFF667EEA), Color(0xFF764BA2)];

  @override
  List<SizePreset> get supportedPresets =>
      const [SizePreset.small, SizePreset.medium];

  @override
  Map<String, dynamic> defaultData() => {'count': 0};

  @override
  Widget buildContent({
    required String instanceId,
    required Map<String, dynamic> data,
    required void Function(Map<String, dynamic>) onUpdate,
  }) {
    final count = data['count'] as int? ?? 0;
    return GestureDetector(
      onTap: () => onUpdate({'count': count + 1}),
      child: Center(
        child: Text('$count', style: const TextStyle(color: Colors.white, fontSize: 32)),
      ),
    );
  }
}
```

### 2. 在 `WidgetPickerSheet` 加入選項

在 `frontend/lib/features/board/widgets/widget_picker_sheet.dart` 的模組清單中加入新模組。

---

## 內建模組

| typeId | label | 支援尺寸 | 說明 |
|--------|-------|----------|------|
| `demo` | 示範 Widget | 全部六種 | 顯示尺寸名稱、格線跨度、instanceId 後 8 碼；供開發驗證佈局 |

---

## 資料持久化

`data` 以 JSON 字串存放於 Hive box `board`，鍵 `widgets`。

- **寫入**：模組呼叫 `onUpdate(newData)` → BLoC 以 `{...oldData, ...newData}` 合併後存入 Hive。
- **讀回**：`BoardBloc` 啟動時（`BoardInitialized`）從 Hive 反序列化，
  過濾掉 registry 中找不到對應 typeId 的項目（避免移除模組後崩潰）。
- **自訂轉換**：若 `data` 包含非 JSON 原生型別（如 `DateTime`、`Color`），
  覆寫 `serialise` / `deserialise` 做轉換：

```dart
@override
Map<String, dynamic> serialise(Map<String, dynamic> data) =>
    {...data, 'targetDate': (data['targetDate'] as DateTime?)?.toIso8601String()};

@override
Map<String, dynamic> deserialise(Map<String, dynamic> raw) =>
    {...raw, 'targetDate': raw['targetDate'] != null
        ? DateTime.parse(raw['targetDate'] as String) : null};
```

---

## 設定面板

覆寫 `buildSettings` 可提供一個於側邊或對話框顯示的設定介面。
回傳 `null`（預設）則看板不顯示設定按鈕。

```dart
@override
Widget? buildSettings({
  required String instanceId,
  required Map<String, dynamic> data,
  required void Function(Map<String, dynamic>) onUpdate,
}) {
  return Column(
    children: [
      TextField(
        onChanged: (v) => onUpdate({'title': v}),
        decoration: const InputDecoration(labelText: '標題'),
      ),
    ],
  );
}
```

---

## 移除回呼

`onRemove(instanceId)` 在 Widget 從看板刪除後立即被呼叫，
適合用來取消訂閱、釋放資源或清除快取：

```dart
@override
void onRemove(String instanceId) {
  _timers[instanceId]?.cancel();
  _timers.remove(instanceId);
}
```

---

## 注意事項

- `typeId` 一旦上線就不能改名，否則已存使用者的 Hive 資料會因找不到對應模組而被丟棄。
- 若需要 `StatefulWidget`（例如計時器、串流），在 `buildContent` 中回傳一個私有的 `StatefulWidget`，不要讓 `WidgetModule` 本身持有狀態。
- `buildContent` 可能在每次 `BoardState` 變更時重建，記得使用 `const` 或適當的 key 避免不必要的 rebuild。
