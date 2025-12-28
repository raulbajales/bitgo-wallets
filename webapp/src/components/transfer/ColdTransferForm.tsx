import React, { useState } from "react";
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Typography,
  TextField,
  Button,
  Alert,
  AlertTitle,
  Autocomplete,
  Chip,
  Switch,
  FormControlLabel,
  Grid,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  CircularProgress,
} from "@mui/material";
import {
  AccountBalanceWallet,
  Security,
  Schedule,
  Warning,
  CheckCircle,
  Info,
  PersonAdd,
} from "@mui/icons-material";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../services/api";

interface ColdTransferRequest {
  destination_address: string;
  amount: string;
  asset: string;
  memo?: string;
  justification: string;
  requestor_user_id: string;
  requires_multiple_signatures: boolean;
  is_priority: boolean;
  additional_approvers?: string[];
}

interface ColdTransferFormProps {
  onSuccess?: () => void;
  onCancel?: () => void;
}

const SUPPORTED_COLD_ASSETS = [
  { symbol: "BTC", name: "Bitcoin" },
  { symbol: "ETH", name: "Ethereum" },
  { symbol: "LTC", name: "Litecoin" },
];

const PRIORITY_REQUIREMENTS = [
  "Amount exceeds $1M USD equivalent",
  "Regulatory compliance requirement",
  "Emergency operational need",
  "Board-approved transaction",
];

export const ColdTransferForm: React.FC<ColdTransferFormProps> = ({
  onSuccess,
  onCancel,
}) => {
  const [formData, setFormData] = useState<ColdTransferRequest>({
    destination_address: "",
    amount: "",
    asset: "",
    memo: "",
    justification: "",
    requestor_user_id: "",
    requires_multiple_signatures: true,
    is_priority: false,
    additional_approvers: [],
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [addressVerified, setAddressVerified] = useState(false);
  const [approverEmail, setApproverEmail] = useState("");

  const queryClient = useQueryClient();

  // Get list of potential approvers
  const { data: approvers, isLoading: loadingApprovers } = useQuery({
    queryKey: ["approvers"],
    queryFn: () =>
      api
        .get("/api/v1/admin/approvers")
        .then((res) => res.data.approvers || []),
  });

  // Cold transfer creation mutation
  const createColdTransferMutation = useMutation({
    mutationFn: (data: ColdTransferRequest) =>
      api.post("/api/v1/transfers/cold", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["transfers"] });
      onSuccess?.();
    },
  });

  // Address verification mutation
  const verifyAddressMutation = useMutation({
    mutationFn: (address: string) =>
      api.post("/api/v1/transfers/verify-address", { address }),
    onSuccess: (response) => {
      setAddressVerified(response.data.valid);
      if (!response.data.valid) {
        setErrors((prev) => ({
          ...prev,
          destination_address: response.data.error || "Invalid address",
        }));
      } else {
        setErrors((prev) => {
          const { destination_address, ...rest } = prev;
          return rest;
        });
      }
    },
  });

  const handleInputChange = (field: keyof ColdTransferRequest, value: any) => {
    setFormData((prev) => ({ ...prev, [field]: value }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors((prev) => {
        const { [field]: _, ...rest } = prev;
        return rest;
      });
    }

    // Reset address verification when address changes
    if (field === "destination_address") {
      setAddressVerified(false);
    }
  };

  const handleVerifyAddress = () => {
    if (formData.destination_address.trim()) {
      verifyAddressMutation.mutate(formData.destination_address.trim());
    }
  };

  const addApprover = () => {
    if (
      approverEmail &&
      !formData.additional_approvers?.includes(approverEmail)
    ) {
      setFormData((prev) => ({
        ...prev,
        additional_approvers: [
          ...(prev.additional_approvers || []),
          approverEmail,
        ],
      }));
      setApproverEmail("");
    }
  };

  const removeApprover = (email: string) => {
    setFormData((prev) => ({
      ...prev,
      additional_approvers:
        prev.additional_approvers?.filter((e) => e !== email) || [],
    }));
  };

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.destination_address.trim()) {
      newErrors.destination_address = "Destination address is required";
    } else if (!addressVerified) {
      newErrors.destination_address = "Please verify the destination address";
    }

    if (!formData.amount.trim() || parseFloat(formData.amount) <= 0) {
      newErrors.amount = "Valid amount is required";
    }

    if (!formData.asset) {
      newErrors.asset = "Asset selection is required";
    }

    if (!formData.justification.trim() || formData.justification.length < 50) {
      newErrors.justification = "Justification must be at least 50 characters";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = () => {
    if (validateForm()) {
      createColdTransferMutation.mutate(formData);
    }
  };

  return (
    <Card>
      <CardHeader
        avatar={<Security color="primary" />}
        title="Cold Storage Transfer Request"
        subheader="Secure, offline custody transfer with manual approval process"
      />
      <CardContent>
        {/* Warning Alert */}
        <Alert severity="warning" sx={{ mb: 3 }}>
          <AlertTitle>Cold Storage Transfer Requirements</AlertTitle>
          <Box component="ul" sx={{ margin: 0, paddingLeft: 2 }}>
            <li>Processing time: 24-72 hours minimum</li>
            <li>Requires manual approval from authorized personnel</li>
            <li>Enhanced security validation and compliance checks</li>
            <li>Cannot be cancelled once initiated</li>
          </Box>
        </Alert>

        <Grid container spacing={3}>
          {/* Transfer Details */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                <AccountBalanceWallet sx={{ mr: 1, verticalAlign: "middle" }} />
                Transfer Details
              </Typography>

              <Box sx={{ mt: 2, space: 2 }}>
                <Autocomplete
                  value={
                    formData.asset
                      ? SUPPORTED_COLD_ASSETS.find(
                          (a) => a.symbol === formData.asset
                        )
                      : null
                  }
                  onChange={(_, value) =>
                    handleInputChange("asset", value?.symbol || "")
                  }
                  options={SUPPORTED_COLD_ASSETS}
                  getOptionLabel={(option) =>
                    `${option.symbol} - ${option.name}`
                  }
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Asset"
                      error={!!errors.asset}
                      helperText={errors.asset}
                      fullWidth
                      margin="normal"
                    />
                  )}
                />

                <TextField
                  label="Amount"
                  type="number"
                  value={formData.amount}
                  onChange={(e) => handleInputChange("amount", e.target.value)}
                  error={!!errors.amount}
                  helperText={errors.amount}
                  fullWidth
                  margin="normal"
                  inputProps={{ step: "0.00000001", min: "0" }}
                />

                <Box sx={{ display: "flex", gap: 1, mt: 2 }}>
                  <TextField
                    label="Destination Address"
                    value={formData.destination_address}
                    onChange={(e) =>
                      handleInputChange("destination_address", e.target.value)
                    }
                    error={!!errors.destination_address}
                    helperText={errors.destination_address}
                    fullWidth
                    multiline
                    rows={2}
                  />
                  <Button
                    variant="outlined"
                    onClick={handleVerifyAddress}
                    disabled={
                      !formData.destination_address.trim() ||
                      verifyAddressMutation.isPending
                    }
                    sx={{ minWidth: 100, alignSelf: "flex-start", mt: 1 }}
                  >
                    {verifyAddressMutation.isPending ? (
                      <CircularProgress size={20} />
                    ) : addressVerified ? (
                      <CheckCircle color="success" />
                    ) : (
                      "Verify"
                    )}
                  </Button>
                </Box>

                <TextField
                  label="Memo (Optional)"
                  value={formData.memo}
                  onChange={(e) => handleInputChange("memo", e.target.value)}
                  fullWidth
                  margin="normal"
                  helperText="Optional memo/tag for the transaction"
                />
              </Box>
            </Paper>
          </Grid>

          {/* Security & Approval */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                <Security sx={{ mr: 1, verticalAlign: "middle" }} />
                Security & Approval
              </Typography>

              <TextField
                label="Business Justification"
                value={formData.justification}
                onChange={(e) =>
                  handleInputChange("justification", e.target.value)
                }
                error={!!errors.justification}
                helperText={
                  errors.justification ||
                  `${formData.justification.length}/50 minimum characters`
                }
                fullWidth
                multiline
                rows={4}
                margin="normal"
                placeholder="Provide detailed business justification for this cold storage transfer..."
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={formData.requires_multiple_signatures}
                    onChange={(e) =>
                      handleInputChange(
                        "requires_multiple_signatures",
                        e.target.checked
                      )
                    }
                  />
                }
                label="Require Multiple Signatures"
                sx={{ mt: 2 }}
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={formData.is_priority}
                    onChange={(e) =>
                      handleInputChange("is_priority", e.target.checked)
                    }
                  />
                }
                label="Priority Request"
                sx={{ mt: 1 }}
              />

              {formData.is_priority && (
                <Alert severity="info" sx={{ mt: 2 }}>
                  <AlertTitle>Priority Request Requirements</AlertTitle>
                  <Typography variant="body2">
                    Priority requests require additional approval. Ensure one of
                    the following applies:
                  </Typography>
                  <List dense>
                    {PRIORITY_REQUIREMENTS.map((req, i) => (
                      <ListItem key={i} sx={{ py: 0 }}>
                        <ListItemIcon>
                          <Info fontSize="small" />
                        </ListItemIcon>
                        <ListItemText primary={req} />
                      </ListItem>
                    ))}
                  </List>
                </Alert>
              )}
            </Paper>
          </Grid>

          {/* Additional Approvers */}
          <Grid item xs={12}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                <PersonAdd sx={{ mr: 1, verticalAlign: "middle" }} />
                Additional Approvers (Optional)
              </Typography>

              <Box sx={{ display: "flex", gap: 1, mb: 2 }}>
                <Autocomplete
                  value={approverEmail}
                  onInputChange={(_, value) => setApproverEmail(value)}
                  options={approvers || []}
                  loading={loadingApprovers}
                  freeSolo
                  sx={{ flexGrow: 1 }}
                  renderInput={(params) => (
                    <TextField
                      {...params}
                      label="Approver Email"
                      placeholder="Add additional approver email..."
                    />
                  )}
                />
                <Button
                  variant="outlined"
                  onClick={addApprover}
                  disabled={!approverEmail}
                >
                  Add
                </Button>
              </Box>

              {formData.additional_approvers &&
                formData.additional_approvers.length > 0 && (
                  <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap" }}>
                    {formData.additional_approvers.map((email) => (
                      <Chip
                        key={email}
                        label={email}
                        onDelete={() => removeApprover(email)}
                        color="primary"
                        variant="outlined"
                      />
                    ))}
                  </Box>
                )}
            </Paper>
          </Grid>

          {/* Processing Timeline */}
          <Grid item xs={12}>
            <Alert severity="info">
              <AlertTitle>
                <Schedule sx={{ mr: 1, verticalAlign: "middle" }} />
                Expected Processing Timeline
              </AlertTitle>
              <List dense>
                <ListItem sx={{ py: 0.5 }}>
                  <ListItemText
                    primary="Initial Review: 2-4 hours"
                    secondary="Compliance and security validation"
                  />
                </ListItem>
                <ListItem sx={{ py: 0.5 }}>
                  <ListItemText
                    primary="Approval Process: 8-24 hours"
                    secondary="Manual approval from authorized personnel"
                  />
                </ListItem>
                <ListItem sx={{ py: 0.5 }}>
                  <ListItemText
                    primary="Execution: 12-48 hours"
                    secondary="Offline signing and blockchain broadcast"
                  />
                </ListItem>
              </List>
            </Alert>
          </Grid>
        </Grid>

        <Divider sx={{ my: 3 }} />

        {/* Action Buttons */}
        <Box sx={{ display: "flex", gap: 2, justifyContent: "flex-end" }}>
          <Button
            variant="outlined"
            onClick={onCancel}
            disabled={createColdTransferMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={createColdTransferMutation.isPending}
            startIcon={
              createColdTransferMutation.isPending ? (
                <CircularProgress size={20} />
              ) : (
                <Security />
              )
            }
          >
            {createColdTransferMutation.isPending
              ? "Creating Request..."
              : "Create Cold Transfer Request"}
          </Button>
        </Box>

        {createColdTransferMutation.isError && (
          <Alert severity="error" sx={{ mt: 2 }}>
            Failed to create cold transfer request:{" "}
            {createColdTransferMutation.error?.message}
          </Alert>
        )}
      </CardContent>
    </Card>
  );
};
