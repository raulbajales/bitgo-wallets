import React, { useState } from "react";
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Alert,
  Toolbar,
  Paper,
  Badge,
  Tooltip,
  Grid,
  Pagination,
  FormControlLabel,
  Switch,
  Divider,
} from "@mui/material";
import {
  Visibility,
  CheckCircle,
  Cancel,
  Warning,
  Schedule,
  Security,
  FilterList,
  Refresh,
  Assignment,
  Priority,
  Info,
} from "@mui/icons-material";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { format, formatDistanceToNow } from "date-fns";
import { api } from "../../services/api";
import {
  ColdTransferTimeline,
  OfflineWorkflowState,
} from "./ColdTransferTimeline";

interface TransferRequest {
  id: string;
  destination_address: string;
  amount: string;
  asset: string;
  status: string;
  created_at: string;
  updated_at: string;
  justification?: string;
  is_priority?: boolean;
  offline_workflow_state?: OfflineWorkflowState;
  workflow_notes?: string;
  sla_breach_at?: string;
  estimated_completion?: string;
  requestor_user_id: string;
}

interface SLASummary {
  total_requests: number;
  sla_breached: number;
  at_risk: number;
  average_processing_time_hours: number;
  priority_requests: number;
}

interface AdminQueueResponse {
  transfers: TransferRequest[];
  count: number;
  sla_summary: SLASummary;
  pagination: {
    limit: number;
    offset: number;
  };
}

interface ColdTransferAdminQueueProps {
  onTransferSelect?: (transfer: TransferRequest) => void;
}

const STATUS_COLORS: Record<
  string,
  "default" | "primary" | "secondary" | "success" | "error" | "info" | "warning"
> = {
  [OfflineWorkflowState.SUBMITTED]: "info",
  [OfflineWorkflowState.COMPLIANCE_REVIEW]: "primary",
  [OfflineWorkflowState.PENDING_APPROVAL]: "warning",
  [OfflineWorkflowState.APPROVED]: "success",
  [OfflineWorkflowState.OFFLINE_SIGNING]: "primary",
  [OfflineWorkflowState.SIGNED]: "success",
  [OfflineWorkflowState.BROADCASTING]: "primary",
  [OfflineWorkflowState.COMPLETED]: "success",
  [OfflineWorkflowState.REJECTED]: "error",
  [OfflineWorkflowState.CANCELLED]: "error",
};

const WORKFLOW_STATE_ACTIONS: Record<
  OfflineWorkflowState,
  OfflineWorkflowState[]
> = {
  [OfflineWorkflowState.SUBMITTED]: [
    OfflineWorkflowState.COMPLIANCE_REVIEW,
    OfflineWorkflowState.REJECTED,
  ],
  [OfflineWorkflowState.COMPLIANCE_REVIEW]: [
    OfflineWorkflowState.PENDING_APPROVAL,
    OfflineWorkflowState.REJECTED,
  ],
  [OfflineWorkflowState.PENDING_APPROVAL]: [
    OfflineWorkflowState.APPROVED,
    OfflineWorkflowState.REJECTED,
  ],
  [OfflineWorkflowState.APPROVED]: [OfflineWorkflowState.OFFLINE_SIGNING],
  [OfflineWorkflowState.OFFLINE_SIGNING]: [OfflineWorkflowState.SIGNED],
  [OfflineWorkflowState.SIGNED]: [OfflineWorkflowState.BROADCASTING],
  [OfflineWorkflowState.BROADCASTING]: [OfflineWorkflowState.COMPLETED],
  [OfflineWorkflowState.COMPLETED]: [],
  [OfflineWorkflowState.REJECTED]: [],
  [OfflineWorkflowState.CANCELLED]: [],
};

export const ColdTransferAdminQueue: React.FC<ColdTransferAdminQueueProps> = ({
  onTransferSelect,
}) => {
  const [page, setPage] = useState(1);
  const [pageSize] = useState(25);
  const [selectedTransfer, setSelectedTransfer] =
    useState<TransferRequest | null>(null);
  const [viewTimelineDialog, setViewTimelineDialog] = useState(false);
  const [stateUpdateDialog, setStateUpdateDialog] = useState(false);
  const [newState, setNewState] = useState<OfflineWorkflowState | "">("");
  const [updateNotes, setUpdateNotes] = useState("");
  const [showSLAOnly, setShowSLAOnly] = useState(false);
  const [showPriorityOnly, setShowPriorityOnly] = useState(false);

  const queryClient = useQueryClient();

  // Fetch cold transfers admin queue
  const { data, isLoading, error, refetch } = useQuery<AdminQueueResponse>({
    queryKey: [
      "cold-transfers-admin",
      page,
      pageSize,
      showSLAOnly,
      showPriorityOnly,
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        limit: pageSize.toString(),
        offset: ((page - 1) * pageSize).toString(),
        ...(showSLAOnly && { sla_only: "true" }),
        ...(showPriorityOnly && { priority_only: "true" }),
      });
      const response = await api.get(
        `/api/v1/transfers/cold/admin-queue?${params}`
      );
      return response.data;
    },
    refetchInterval: 30000, // Auto-refresh every 30 seconds
  });

  // Update workflow state mutation
  const updateStateMutation = useMutation({
    mutationFn: async ({
      id,
      state,
      notes,
    }: {
      id: string;
      state: OfflineWorkflowState;
      notes: string;
    }) => {
      return api.put(`/api/v1/transfers/${id}/offline-workflow-state`, {
        state,
        notes,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cold-transfers-admin"] });
      setStateUpdateDialog(false);
      setSelectedTransfer(null);
      setNewState("");
      setUpdateNotes("");
    },
  });

  const handleViewTimeline = (transfer: TransferRequest) => {
    setSelectedTransfer(transfer);
    setViewTimelineDialog(true);
    onTransferSelect?.(transfer);
  };

  const handleUpdateState = (transfer: TransferRequest) => {
    setSelectedTransfer(transfer);
    setStateUpdateDialog(true);
  };

  const handleStateUpdate = () => {
    if (selectedTransfer && newState) {
      updateStateMutation.mutate({
        id: selectedTransfer.id,
        state: newState as OfflineWorkflowState,
        notes: updateNotes,
      });
    }
  };

  const getAvailableActions = (currentState?: OfflineWorkflowState) => {
    if (!currentState) return [];
    return WORKFLOW_STATE_ACTIONS[currentState] || [];
  };

  const isSLAAtRisk = (transfer: TransferRequest) => {
    return (
      transfer.sla_breach_at && new Date(transfer.sla_breach_at) < new Date()
    );
  };

  const sla = data?.sla_summary;

  return (
    <Box>
      {/* SLA Summary Dashboard */}
      {sla && (
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Total Requests
                </Typography>
                <Typography variant="h4">{sla.total_requests}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  SLA Breached
                </Typography>
                <Typography variant="h4" color="error.main">
                  <Badge badgeContent={sla.sla_breached} color="error">
                    {sla.sla_breached}
                  </Badge>
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  At Risk
                </Typography>
                <Typography variant="h4" color="warning.main">
                  {sla.at_risk}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Avg Processing (hrs)
                </Typography>
                <Typography variant="h4">
                  {sla.average_processing_time_hours.toFixed(1)}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Main Queue Card */}
      <Card>
        <CardHeader
          avatar={<Security color="primary" />}
          title="Cold Storage Transfer Queue"
          subheader={`${data?.count || 0} requests pending attention`}
          action={
            <Box sx={{ display: "flex", gap: 1 }}>
              <Tooltip title="Refresh">
                <IconButton onClick={() => refetch()}>
                  <Refresh />
                </IconButton>
              </Tooltip>
            </Box>
          }
        />

        {/* Filters */}
        <Box sx={{ px: 2, pb: 2 }}>
          <Toolbar variant="dense" sx={{ px: 0, minHeight: "auto" }}>
            <FilterList sx={{ mr: 2 }} />
            <FormControlLabel
              control={
                <Switch
                  checked={showSLAOnly}
                  onChange={(e) => setShowSLAOnly(e.target.checked)}
                  size="small"
                />
              }
              label="SLA Risk Only"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={showPriorityOnly}
                  onChange={(e) => setShowPriorityOnly(e.target.checked)}
                  size="small"
                />
              }
              label="Priority Only"
              sx={{ ml: 2 }}
            />
          </Toolbar>
        </Box>

        <CardContent sx={{ pt: 0 }}>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              Failed to load cold transfer queue: {error.message}
            </Alert>
          )}

          <TableContainer component={Paper} variant="outlined">
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Request Details</TableCell>
                  <TableCell>Amount</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>SLA Status</TableCell>
                  <TableCell align="center">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {isLoading ? (
                  <TableRow>
                    <TableCell colSpan={6} align="center">
                      <Typography>Loading transfers...</Typography>
                    </TableCell>
                  </TableRow>
                ) : data?.transfers.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} align="center">
                      <Typography color="textSecondary">
                        No cold transfer requests found
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  data?.transfers.map((transfer) => (
                    <TableRow key={transfer.id} hover>
                      <TableCell>
                        <Box>
                          <Typography variant="body2" fontWeight="medium">
                            {transfer.destination_address.slice(0, 20)}...
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            ID: {transfer.id.slice(0, 8)}...
                          </Typography>
                          {transfer.is_priority && (
                            <Chip
                              label="Priority"
                              size="small"
                              color="warning"
                              icon={<Priority />}
                              sx={{ ml: 1 }}
                            />
                          )}
                        </Box>
                      </TableCell>

                      <TableCell>
                        <Typography variant="body2" fontWeight="medium">
                          {transfer.amount} {transfer.asset}
                        </Typography>
                      </TableCell>

                      <TableCell>
                        <Chip
                          label={transfer.offline_workflow_state || "SUBMITTED"}
                          color={
                            STATUS_COLORS[
                              transfer.offline_workflow_state ||
                                OfflineWorkflowState.SUBMITTED
                            ]
                          }
                          size="small"
                        />
                      </TableCell>

                      <TableCell>
                        <Typography variant="body2">
                          {format(
                            new Date(transfer.created_at),
                            "MMM dd, HH:mm"
                          )}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {formatDistanceToNow(new Date(transfer.created_at))}{" "}
                          ago
                        </Typography>
                      </TableCell>

                      <TableCell>
                        {isSLAAtRisk(transfer) ? (
                          <Chip
                            label="SLA Breach"
                            color="error"
                            size="small"
                            icon={<Warning />}
                          />
                        ) : transfer.sla_breach_at ? (
                          <Chip
                            label="On Track"
                            color="success"
                            size="small"
                            icon={<Schedule />}
                          />
                        ) : (
                          <Typography variant="caption" color="text.secondary">
                            No SLA
                          </Typography>
                        )}
                      </TableCell>

                      <TableCell align="center">
                        <Box
                          sx={{
                            display: "flex",
                            gap: 1,
                            justifyContent: "center",
                          }}
                        >
                          <Tooltip title="View Timeline">
                            <IconButton
                              size="small"
                              onClick={() => handleViewTimeline(transfer)}
                            >
                              <Visibility />
                            </IconButton>
                          </Tooltip>

                          {getAvailableActions(transfer.offline_workflow_state)
                            .length > 0 && (
                            <Tooltip title="Update Status">
                              <IconButton
                                size="small"
                                color="primary"
                                onClick={() => handleUpdateState(transfer)}
                              >
                                <Assignment />
                              </IconButton>
                            </Tooltip>
                          )}
                        </Box>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>

          {/* Pagination */}
          {data && data.count > pageSize && (
            <Box sx={{ display: "flex", justifyContent: "center", mt: 2 }}>
              <Pagination
                count={Math.ceil(data.count / pageSize)}
                page={page}
                onChange={(_, value) => setPage(value)}
                color="primary"
              />
            </Box>
          )}
        </CardContent>
      </Card>

      {/* Timeline View Dialog */}
      <Dialog
        open={viewTimelineDialog}
        onClose={() => setViewTimelineDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Transfer Timeline
          {selectedTransfer && (
            <Typography variant="subtitle2" color="text.secondary">
              {selectedTransfer.amount} {selectedTransfer.asset} to{" "}
              {selectedTransfer.destination_address.slice(0, 20)}...
            </Typography>
          )}
        </DialogTitle>
        <DialogContent>
          {selectedTransfer && (
            <ColdTransferTimeline
              transfer={selectedTransfer}
              showSLAWarnings={true}
            />
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewTimelineDialog(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* State Update Dialog */}
      <Dialog
        open={stateUpdateDialog}
        onClose={() => setStateUpdateDialog(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Update Workflow State</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 1 }}>
            {selectedTransfer && (
              <Alert severity="info" sx={{ mb: 2 }}>
                <Typography variant="subtitle2">
                  Current State:{" "}
                  {selectedTransfer.offline_workflow_state || "SUBMITTED"}
                </Typography>
                <Typography variant="body2">
                  {selectedTransfer.amount} {selectedTransfer.asset}
                </Typography>
              </Alert>
            )}

            <FormControl fullWidth sx={{ mb: 2 }}>
              <InputLabel>New State</InputLabel>
              <Select
                value={newState}
                onChange={(e) =>
                  setNewState(e.target.value as OfflineWorkflowState)
                }
                label="New State"
              >
                {getAvailableActions(
                  selectedTransfer?.offline_workflow_state
                ).map((state) => (
                  <MenuItem key={state} value={state}>
                    {state.replace(/_/g, " ")}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <TextField
              label="Notes (Optional)"
              value={updateNotes}
              onChange={(e) => setUpdateNotes(e.target.value)}
              multiline
              rows={3}
              fullWidth
              placeholder="Add any relevant notes about this state change..."
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setStateUpdateDialog(false)}
            disabled={updateStateMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleStateUpdate}
            variant="contained"
            disabled={!newState || updateStateMutation.isPending}
          >
            {updateStateMutation.isPending ? "Updating..." : "Update State"}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
