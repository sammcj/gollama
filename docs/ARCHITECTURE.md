# Gollama Architecture

## Model Update Process

### Current Implementation

The current model update process in `editModelfile` follows these steps:

1. Fetch the current modelfile from the server
2. Open the modelfile in an editor for modification
3. Read the edited content
4. Send a Create request to update the model

### Issue: "Unknown Type" Error

When updating models using the `e` command and saving changes to the Modelfile, users may encounter an error:
```
Error updating model: error updating model with new modelfile: unknown type
```

This error occurs because:

1. The model may still be loaded in memory when the update is attempted
2. The modelfile content may not be properly validated before sending to the Ollama API
3. The Create request may need additional parameters or headers

### Proposed Solution

To resolve this issue, the model update process should be modified to:

1. Unload the model before attempting the update:
   ```go
   // First unload the model
   _, err = unloadModel(client, modelName)
   if err != nil {
       return "", fmt.Errorf("error unloading model before update: %v", err)
   }
   ```

2. Validate the modelfile content:
   - Ensure it contains required fields (FROM, etc.)
   - Check for proper formatting
   - Validate any parameters

3. Send the Create request with proper headers and parameters:
   ```go
   createReq := &api.CreateRequest{
       Model: modelName,
       Files: map[string]string{
           "modelfile": string(newModelfileContent),
       },
   }
   ```

### Implementation Details

The `editModelfile` function in `operations.go` should be updated to include these specific steps:

1. Add a validation step for the modelfile content:
   ```go
   // Validate modelfile content
   if !strings.Contains(string(newModelfileContent), "FROM") {
       return "", fmt.Errorf("invalid modelfile: missing FROM directive")
   }
   ```

2. Unload the model before attempting the update:
   ```go
   // Unload the model first
   if _, err := unloadModel(client, modelName); err != nil {
       logging.DebugLogger.Printf("Error unloading model %s: %v\n", modelName, err)
       // Continue anyway as the error might be because the model wasn't loaded
   }
   ```

3. Improve error handling with specific error messages:
   ```go
   if err := client.Create(ctx, createReq, progressCallback); err != nil {
       if strings.Contains(err.Error(), "unknown type") {
           return "", fmt.Errorf("error updating model: invalid modelfile format or content")
       }
       return "", fmt.Errorf("error updating model: %v", err)
   }
   ```

4. Add proper logging for debugging:
   ```go
   logging.DebugLogger.Printf("Updating model %s with new modelfile\n", modelName)
   logging.DebugLogger.Printf("Modelfile content:\n%s\n", newModelfileContent)
   ```

This implementation will provide:
- Better error messages that explain what went wrong
- Proper model unloading before updates
- Validation of modelfile content
- Detailed logging for debugging issues

To implement these changes, you'll need to:
1. Switch to Code mode
2. Update the `editModelfile` function in `operations.go`
3. Test the changes with various model updates

The improved implementation will make model updates more reliable and provide better feedback when issues occur.
