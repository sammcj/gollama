# Implementation Plan

- [x] 1. Fix Wails v3 service registration and method exposure
  - Update `gui/app.go` to properly expose service methods to JavaScript runtime
  - Implement direct method binding on App struct for Wails v3 compatibility
  - Add comprehensive logging to debug service initialisation and method registration
  - _Requirements: 1.1, 1.4, 4.1, 4.3_

- [x] 2. Create robust JavaScript API layer with error handling
  - Implement `GollamaAPI` class with multiple fallback patterns for method discovery
  - Add runtime method detection and logging to identify available Wails methods
  - Create comprehensive error handling with user-friendly toast notifications
  - _Requirements: 1.2, 1.3, 3.1, 3.2_

- [x] 3. Implement data transfer objects and conversion functions
  - Create `ModelDTO`, `ConfigDTO`, `VRAMEstimationDTO` and other DTOs for JavaScript communication
  - Implement conversion functions from core models to DTOs with proper formatting
  - Add JSON serialisation tags and validation for all DTO structures
  - _Requirements: 1.2, 2.1, 2.3_

- [x] 4. Add service method implementations on App struct
  - Implement `GetModels()`, `DeleteModel()`, `RunModel()` and other core methods
  - Add proper error handling and logging for each service method
  - Ensure all methods return appropriate DTOs for JavaScript consumption
  - _Requirements: 1.1, 1.2, 2.2, 3.3_

- [x] 5. Enhance JavaScript frontend integration
  - Update existing JavaScript code to use new `GollamaAPI` class
  - Replace direct `window.wails` calls with API wrapper methods
  - Add proper error handling and user feedback for all service calls
  - _Requirements: 1.3, 2.2, 2.4, 3.1_

- [x] 6. Create unit tests for the new GUI
  - Write unit tests for service binding and method exposure
  - Add comprehensive test coverage for all service methods and error handling
  - Create mock service for testing without Ollama dependency
  - _Requirements: 3.3, 3.4, 4.4_

- [x] 7. Add debugging and diagnostic capabilities for the GUI
  - Implement runtime method discovery and logging in JavaScript
  - Add service health check endpoint with detailed status information
  - Create diagnostic tools to verify service binding is working correctly
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 8. Optimise performance and finalise integration of the GUI
  - Optimise service call performance and reduce latency
  - Add caching for frequently accessed data like model lists
  - Finalise error messages and user feedback mechanisms
  - _Requirements: 2.3, 2.4, 4.2, 4.4_

- [ ] 9. Fix CSS compilation to include custom component styles
  - The Tailwind CSS build process is not including custom component styles from input.css
  - Update the build process to properly compile custom @layer components styles
  - Ensure btn-primary, nav-link, and other custom classes are included in compiled CSS
  - Verify button styling and functionality work correctly in the GUI
  - _Requirements: 2.4, 4.2_
