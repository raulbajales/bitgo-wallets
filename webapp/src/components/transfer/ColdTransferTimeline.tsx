import React from "react";
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Typography,
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent,
  Chip,
  Alert,
  LinearProgress,
  Grid,
  Paper,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Divider,
} from "@mui/material";
import {
  CheckCircle,
  Schedule,
  HourglassEmpty,
  Security,
  AccountBalanceWallet,
  Warning,
  Assignment,
  VpnKey,
  Broadcast,
  Person,
  AccessTime,
  Info,
} from "@mui/icons-material";
import { format, formatDistanceToNow } from "date-fns";

export enum OfflineWorkflowState {
  SUBMITTED = "SUBMITTED",
  COMPLIANCE_REVIEW = "COMPLIANCE_REVIEW",
  PENDING_APPROVAL = "PENDING_APPROVAL",
  APPROVED = "APPROVED",
  OFFLINE_SIGNING = "OFFLINE_SIGNING",
  SIGNED = "SIGNED",
  BROADCASTING = "BROADCASTING",
  COMPLETED = "COMPLETED",
  REJECTED = "REJECTED",
  CANCELLED = "CANCELLED",
}

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
}

interface WorkflowStep {
  state: OfflineWorkflowState;
  title: string;
  description: string;
  icon: React.ReactElement;
  estimatedDuration: string;
  isCompleted?: boolean;
  isCurrent?: boolean;
  completedAt?: string;
  notes?: string;
}

interface ColdTransferTimelineProps {
  transfer: TransferRequest;
  showSLAWarnings?: boolean;
}

const getWorkflowSteps = (transfer: TransferRequest): WorkflowStep[] => {
  const currentState =
    transfer.offline_workflow_state || OfflineWorkflowState.SUBMITTED;

  const steps: WorkflowStep[] = [
    {
      state: OfflineWorkflowState.SUBMITTED,
      title: "Request Submitted",
      description: "Cold transfer request received and queued for review",
      icon: <Assignment />,
      estimatedDuration: "Immediate",
      isCompleted: true,
      completedAt: transfer.created_at,
    },
    {
      state: OfflineWorkflowState.COMPLIANCE_REVIEW,
      title: "Compliance Review",
      description: "Automated compliance checks and validation",
      icon: <Security />,
      estimatedDuration: "1-2 hours",
    },
    {
      state: OfflineWorkflowState.PENDING_APPROVAL,
      title: "Pending Approval",
      description: "Awaiting manual approval from authorized personnel",
      icon: <Person />,
      estimatedDuration: "4-24 hours",
    },
    {
      state: OfflineWorkflowState.APPROVED,
      title: "Approved",
      description: "Transfer approved and queued for offline signing",
      icon: <CheckCircle />,
      estimatedDuration: "2-4 hours",
    },
    {
      state: OfflineWorkflowState.OFFLINE_SIGNING,
      title: "Offline Signing",
      description: "Transaction being signed in secure offline environment",
      icon: <VpnKey />,
      estimatedDuration: "8-24 hours",
    },
    {
      state: OfflineWorkflowState.SIGNED,
      title: "Signed",
      description: "Transaction signed and ready for broadcast",
      icon: <Security />,
      estimatedDuration: "1-2 hours",
    },
    {
      state: OfflineWorkflowState.BROADCASTING,
      title: "Broadcasting",
      description: "Transaction being broadcast to the network",
      icon: <Broadcast />,
      estimatedDuration: "5-30 minutes",
    },
    {
      state: OfflineWorkflowState.COMPLETED,
      title: "Completed",
      description: "Transfer completed and confirmed on blockchain",
      icon: <CheckCircle />,
      estimatedDuration: "Complete",
    },
  ];

  // Mark steps as completed/current based on current state
  const currentIndex = steps.findIndex((step) => step.state === currentState);
  steps.forEach((step, index) => {
    step.isCompleted = index < currentIndex;
    step.isCurrent = index === currentIndex;
  });

  return steps;
};

const getStatusColor = (state?: OfflineWorkflowState) => {
  switch (state) {
    case OfflineWorkflowState.COMPLETED:
      return "success";
    case OfflineWorkflowState.REJECTED:
    case OfflineWorkflowState.CANCELLED:
      return "error";
    case OfflineWorkflowState.PENDING_APPROVAL:
    case OfflineWorkflowState.OFFLINE_SIGNING:
      return "warning";
    default:
      return "primary";
  }
};

export const ColdTransferTimeline: React.FC<ColdTransferTimelineProps> = ({
  transfer,
  showSLAWarnings = true,
}) => {
  const workflowSteps = getWorkflowSteps(transfer);
  const currentStep = workflowSteps.find((step) => step.isCurrent);
  const completedSteps = workflowSteps.filter(
    (step) => step.isCompleted
  ).length;
  const totalSteps = workflowSteps.length - 1; // Exclude completed state for progress calc
  const progressPercentage = (completedSteps / totalSteps) * 100;

  const isSLAAtRisk =
    transfer.sla_breach_at && new Date(transfer.sla_breach_at) < new Date();
  const estimatedCompletion = transfer.estimated_completion
    ? new Date(transfer.estimated_completion)
    : null;

  return (
    <Box>
      {/* Header with Transfer Info */}
      <Card sx={{ mb: 3 }}>
        <CardHeader
          avatar={<AccountBalanceWallet color="primary" />}
          title={`Cold Transfer: ${transfer.amount} ${transfer.asset}`}
          subheader={`To: ${transfer.destination_address.slice(
            0,
            20
          )}...${transfer.destination_address.slice(-10)}`}
          action={
            <Chip
              label={transfer.offline_workflow_state || "SUBMITTED"}
              color={getStatusColor(transfer.offline_workflow_state)}
              variant="outlined"
            />
          }
        />
        <CardContent>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Box sx={{ mb: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  Progress
                </Typography>
                <LinearProgress
                  variant="determinate"
                  value={progressPercentage}
                  sx={{ mt: 1, height: 8, borderRadius: 4 }}
                />
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ mt: 0.5 }}
                >
                  {completedSteps} of {totalSteps} steps completed
                </Typography>
              </Box>

              {currentStep && (
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Current Step
                  </Typography>
                  <Typography variant="h6">{currentStep.title}</Typography>
                  <Typography variant="body2" color="text.secondary">
                    {currentStep.description}
                  </Typography>
                </Box>
              )}
            </Grid>

            <Grid item xs={12} md={6}>
              <List dense>
                <ListItem sx={{ px: 0 }}>
                  <ListItemIcon>
                    <AccessTime fontSize="small" />
                  </ListItemIcon>
                  <ListItemText
                    primary="Created"
                    secondary={format(
                      new Date(transfer.created_at),
                      "MMM dd, yyyy HH:mm"
                    )}
                  />
                </ListItem>
                <ListItem sx={{ px: 0 }}>
                  <ListItemIcon>
                    <AccessTime fontSize="small" />
                  </ListItemIcon>
                  <ListItemText
                    primary="Last Updated"
                    secondary={`${format(
                      new Date(transfer.updated_at),
                      "MMM dd, yyyy HH:mm"
                    )} (${formatDistanceToNow(
                      new Date(transfer.updated_at)
                    )} ago)`}
                  />
                </ListItem>
                {estimatedCompletion && (
                  <ListItem sx={{ px: 0 }}>
                    <ListItemIcon>
                      <Schedule fontSize="small" />
                    </ListItemIcon>
                    <ListItemText
                      primary="Estimated Completion"
                      secondary={format(
                        estimatedCompletion,
                        "MMM dd, yyyy HH:mm"
                      )}
                    />
                  </ListItem>
                )}
              </List>
            </Grid>
          </Grid>

          {transfer.is_priority && (
            <Chip
              label="Priority Request"
              color="warning"
              icon={<Warning />}
              size="small"
              sx={{ mt: 2 }}
            />
          )}
        </CardContent>
      </Card>

      {/* SLA Warnings */}
      {showSLAWarnings && isSLAAtRisk && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          <Typography variant="subtitle2">SLA Warning</Typography>
          This transfer is approaching or has exceeded the expected processing
          time.
          {transfer.sla_breach_at && (
            <Typography variant="body2" sx={{ mt: 1 }}>
              Expected completion was:{" "}
              {format(new Date(transfer.sla_breach_at), "MMM dd, yyyy HH:mm")}
            </Typography>
          )}
        </Alert>
      )}

      {/* Timeline */}
      <Card>
        <CardHeader
          title="Workflow Timeline"
          subheader="Detailed progress through the cold storage transfer process"
        />
        <CardContent>
          <Timeline position="right">
            {workflowSteps.map((step, index) => (
              <TimelineItem key={step.state}>
                <TimelineOppositeContent
                  sx={{ m: "auto 0" }}
                  variant="body2"
                  color="text.secondary"
                >
                  {step.estimatedDuration}
                  {step.completedAt && (
                    <Typography variant="caption" display="block">
                      {format(new Date(step.completedAt), "MMM dd, HH:mm")}
                    </Typography>
                  )}
                </TimelineOppositeContent>

                <TimelineSeparator>
                  <TimelineDot
                    color={
                      step.isCompleted
                        ? "success"
                        : step.isCurrent
                        ? "primary"
                        : "grey"
                    }
                    variant={step.isCurrent ? "outlined" : "filled"}
                  >
                    {step.isCompleted ? (
                      <CheckCircle />
                    ) : step.isCurrent ? (
                      <HourglassEmpty />
                    ) : (
                      step.icon
                    )}
                  </TimelineDot>
                  {index < workflowSteps.length - 1 && <TimelineConnector />}
                </TimelineSeparator>

                <TimelineContent sx={{ py: "12px", px: 2 }}>
                  <Typography variant="h6" component="span">
                    {step.title}
                  </Typography>
                  <Typography color="text.secondary">
                    {step.description}
                  </Typography>

                  {step.notes && (
                    <Paper
                      variant="outlined"
                      sx={{ p: 1, mt: 1, backgroundColor: "action.hover" }}
                    >
                      <Typography variant="body2">
                        <Info
                          fontSize="small"
                          sx={{ mr: 1, verticalAlign: "middle" }}
                        />
                        {step.notes}
                      </Typography>
                    </Paper>
                  )}

                  {step.isCurrent &&
                    currentStep?.estimatedDuration !== "Immediate" && (
                      <Chip
                        label="In Progress"
                        color="primary"
                        size="small"
                        sx={{ mt: 1 }}
                        icon={<HourglassEmpty />}
                      />
                    )}
                </TimelineContent>
              </TimelineItem>
            ))}
          </Timeline>
        </CardContent>
      </Card>

      {/* Business Justification */}
      {transfer.justification && (
        <Card sx={{ mt: 3 }}>
          <CardHeader title="Business Justification" />
          <CardContent>
            <Typography variant="body1">{transfer.justification}</Typography>
          </CardContent>
        </Card>
      )}
    </Box>
  );
};
