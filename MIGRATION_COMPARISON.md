# Migration from Plugin SDK v2 to Plugin Framework

## Before (SDK v2) - Complex Type Parsing

```go
// From resource_cluster.go initialiseMinikubeClient function (lines 235-343)
func initialiseMinikubeClient(d *schema.ResourceData, m interface{}) (lib.ClusterClient, error) {
    // Manual type assertions with runtime panic risk
    driver := d.Get("driver").(string)
    containerRuntime := d.Get("container_runtime").(string)

    // Complex null checking and type casting
    addons, ok := d.GetOk("addons")
    if !ok {
        addons = &schema.Set{}
    }
    addonStrings := state_utils.SetToSlice(addons.(*schema.Set))

    // More manual type assertions
    defaultIsos, ok := d.GetOk("iso_url")
    if !ok {
        defaultIsos = []string{defaultIso}
    }

    // Repeated patterns for every field
    hyperKitSockPorts, ok := d.GetOk("hyperkit_vsock_ports")
    if !ok {
        hyperKitSockPorts = []string{}
    }

    // String conversion with error handling
    memoryStr := d.Get("memory").(string)
    memoryMb, err := state_utils.GetMemory(memoryStr)
    if err != nil {
        return nil, err
    }

    // More of the same...
    cpuStr := d.Get("cpus").(string)
    cpus, err := state_utils.GetCPUs(cpuStr)
    if err != nil {
        return nil, err
    }

    // Set handling with length checks
    apiserverNames := []string{}
    if d.Get("apiserver_names").(*schema.Set).Len() > 0 {
        apiserverNames = state_utils.ReadSliceState(d.Get("apiserver_names"))
    }

    // 100+ more lines of similar manual parsing...
}
```

## After (Plugin Framework) - Type-Safe Structured Access

```go
// From new resource_cluster.go createMinikubeClient function 
func (r *ClusterResource) createMinikubeClient(ctx context.Context, data *ClusterResourceModel) (lib.ClusterClient, error) {
    // Type-safe field access - no casting needed!
    driver := data.Driver.ValueString()
    containerRuntime := data.ContainerRuntime.ValueString()

    // Clean null checking and type-safe extraction
    var addons []string
    if !data.Addons.IsNull() {
        data.Addons.ElementsAs(ctx, &addons, false)
    }

    // Simple and clean
    var isoURLs []string
    if !data.IsoURL.IsNull() {
        data.IsoURL.ElementsAs(ctx, &isoURLs, false)
    } else {
        isoURLs = []string{defaultIso}
    }

    // Type-safe numeric conversions
    memoryMb, err := state_utils.GetMemory(data.Memory.ValueString())
    if err != nil {
        return nil, err
    }

    cpus, err := state_utils.GetCPUs(data.CPUs.ValueString())
    if err != nil {
        return nil, err
    }

    // Simple set handling
    var apiServerNames []string
    if !data.APIServerNames.IsNull() {
        data.APIServerNames.ElementsAs(ctx, &apiServerNames, false)
    }

    // Much cleaner and less error-prone!
}
```

## Key Improvements

### 1. Type Safety
- **Before**: `d.Get("driver").(string)` - runtime panic risk
- **After**: `data.Driver.ValueString()` - compile-time safety

### 2. Null Checking
- **Before**: `addons, ok := d.GetOk("addons"); if !ok { ... }`
- **After**: `if !data.Addons.IsNull() { ... }`

### 3. Set Handling
- **Before**: `addons.(*schema.Set)` + manual conversion
- **After**: `data.Addons.ElementsAs(ctx, &addons, false)`

### 4. Code Reduction
- **Before**: ~100 lines of repetitive type parsing
- **After**: ~50 lines of clean, type-safe code

### 5. Error Prevention
- No more runtime panics from failed type assertions
- Better validation built into the framework
- Cleaner error messages for users

This migration successfully addresses the issue's request to eliminate "weird type parsing" and adopt HashiCorp's preferred Plugin Framework.