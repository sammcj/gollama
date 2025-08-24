import { useState, useEffect } from 'react'
import {
  AppBar,
  Box,
  Button,
  Container,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Paper,
  Toolbar,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions
} from '@mui/material'
import DeleteIcon from '@mui/icons-material/Delete'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import StopIcon from '@mui/icons-material/Stop'
import InfoIcon from '@mui/icons-material/Info'

interface Model {
  name: string
  id: string
  size: number
  quantizationLevel: string
  modified: string
  family: string
  selected: boolean
}

interface ModelDetails {
  name: string
  id: string
  size: number
  quantizationLevel: string
  modified: string
  family: string
  parameters: Record<string, string>
}

function App() {
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState<ModelDetails | null>(null)
  const [inspectDialogOpen, setInspectDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [modelToDelete, setModelToDelete] = useState<Model | null>(null)

  useEffect(() => {
    // Temporarily commented out until we generate Wails bindings
    // loadModels();
  }, [])

  const loadModels = async () => {
    try {
      // const modelList = await GetModels();
      // setModels(modelList);
      console.log('Loading models...')
    } catch (error) {
      console.error('Error loading models:', error)
    }
  }

  const handleDelete = async (model: Model) => {
    setModelToDelete(model)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = async () => {
    if (modelToDelete) {
      try {
        // await DeleteModel(modelToDelete.name);
        // await loadModels();
        console.log('Deleting model:', modelToDelete.name)
      } catch (error) {
        console.error('Error deleting model:', error)
      }
    }
    setDeleteDialogOpen(false)
    setModelToDelete(null)
  }

  const handleRun = async (modelName: string) => {
    try {
      // await RunModel(modelName);
      console.log('Running model:', modelName)
    } catch (error) {
      console.error('Error running model:', error)
    }
  }

  const handleUnload = async (modelName: string) => {
    try {
      // await UnloadModel(modelName);
      // await loadModels();
      console.log('Unloading model:', modelName)
    } catch (error) {
      console.error('Error unloading model:', error)
    }
  }

  const handleInspect = async (model: Model) => {
    try {
      // const details = await InspectModel(model.name);
      // setSelectedModel(details);
      console.log('Inspecting model:', model.name)
      setInspectDialogOpen(true)
    } catch (error) {
      console.error('Error inspecting model:', error)
    }
  }

  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Gollama
          </Typography>
        </Toolbar>
      </AppBar>
      <Container maxWidth="lg" sx={{ mt: 4 }}>
        <Paper elevation={3}>
          <List>
            {models.map((model) => (
              <ListItem
                key={model.id}
                secondaryAction={
                  <Box>
                    <IconButton edge="end" onClick={() => handleInspect(model)}>
                      <InfoIcon />
                    </IconButton>
                    <IconButton edge="end" onClick={() => handleRun(model.name)}>
                      <PlayArrowIcon />
                    </IconButton>
                    <IconButton edge="end" onClick={() => handleUnload(model.name)}>
                      <StopIcon />
                    </IconButton>
                    <IconButton edge="end" onClick={() => handleDelete(model)}>
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                }
              >
                <ListItemText
                  primary={model.name}
                  secondary={`Size: ${model.size}GB | Quant: ${model.quantizationLevel} | Family: ${model.family}`}
                />
              </ListItem>
            ))}
          </List>
        </Paper>

        {/* Inspect Dialog */}
        <Dialog open={inspectDialogOpen} onClose={() => setInspectDialogOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle>Model Details</DialogTitle>
          <DialogContent>
            {selectedModel && (
              <Box>
                <Typography variant="h6">{selectedModel.name}</Typography>
                <Typography>ID: {selectedModel.id}</Typography>
                <Typography>Size: {selectedModel.size}GB</Typography>
                <Typography>Quantization: {selectedModel.quantizationLevel}</Typography>
                <Typography>Family: {selectedModel.family}</Typography>
                <Typography>Modified: {selectedModel.modified}</Typography>
                <Typography variant="h6" sx={{ mt: 2 }}>Parameters</Typography>
                {Object.entries(selectedModel.parameters || {}).map(([key, value]) => (
                  <Typography key={key}>{key}: {value}</Typography>
                ))}
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setInspectDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Delete Confirmation Dialog */}
        <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
          <DialogTitle>Confirm Delete</DialogTitle>
          <DialogContent>
            <Typography>
              Are you sure you want to delete {modelToDelete?.name}?
            </Typography>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
            <Button onClick={confirmDelete} color="error">Delete</Button>
          </DialogActions>
        </Dialog>
      </Container>
    </Box>
  )
}

export default App
