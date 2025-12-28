import React from "react";
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider, createTheme } from "@mui/material/styles";
import {
  CssBaseline,
  Container,
  Box,
  AppBar,
  Toolbar,
  Typography,
} from "@mui/material";

import { ColdTransferForm } from "./components/transfer/ColdTransferForm";
import { ColdTransferAdminQueue } from "./components/admin/ColdTransferAdminQueue";

// Create a theme
const theme = createTheme({
  palette: {
    mode: "light",
    primary: {
      main: "#1976d2",
    },
    secondary: {
      main: "#dc004e",
    },
  },
});

// Create a client for React Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Router>
          <Box sx={{ flexGrow: 1 }}>
            <AppBar position="static">
              <Toolbar>
                <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
                  BitGo Wallets - Cold Storage
                </Typography>
              </Toolbar>
            </AppBar>

            <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
              <Routes>
                <Route
                  path="/"
                  element={<Navigate to="/cold-transfer" replace />}
                />
                <Route
                  path="/cold-transfer"
                  element={
                    <ColdTransferForm
                      onSuccess={() =>
                        console.log("Transfer created successfully")
                      }
                      onCancel={() => console.log("Transfer cancelled")}
                    />
                  }
                />
                <Route path="/admin" element={<ColdTransferAdminQueue />} />
              </Routes>
            </Container>
          </Box>
        </Router>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
