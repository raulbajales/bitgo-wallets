"use client";

import React, { useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { formatCurrency, truncateAddress } from "@/lib/utils";
import { type Wallet } from "@/components/wallets/wallet-card";

interface CreateTransferFormProps {
  wallet: Wallet;
  onSubmit: (transferData: TransferFormData) => Promise<void>;
  onCancel: () => void;
}

export interface TransferFormData {
  recipientAddress: string;
  amountString: string;
  coin: string;
  transferType: "custodial" | "hot" | "cold" | "warm";
  memo?: string;

  // Additional fields for warm/cold transfers
  businessPurpose?: string;
  requestorName?: string;
  requestorEmail?: string;
  urgencyLevel?: "low" | "normal" | "high" | "critical";
  autoProcess?: boolean; // For warm transfers
}

interface FormErrors {
  recipientAddress?: string;
  amountString?: string;
  memo?: string;
  general?: string;
}

export function CreateTransferForm({
  wallet,
  onSubmit,
  onCancel,
}: CreateTransferFormProps) {
  const [formData, setFormData] = useState<TransferFormData>({
    recipientAddress: "",
    amountString: "",
    coin: wallet.coin,
    transferType: wallet.walletType as "custodial" | "hot" | "cold" | "warm",
    memo: "",
    businessPurpose: "",
    requestorName: "",
    requestorEmail: "",
    urgencyLevel: "normal",
    autoProcess: wallet.walletType === "warm", // Default to auto for warm wallets
  });

  const [errors, setErrors] = useState<FormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Validate form data
  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    // Validate recipient address
    if (!formData.recipientAddress.trim()) {
      newErrors.recipientAddress = "Recipient address is required";
    } else if (formData.recipientAddress.length < 10) {
      newErrors.recipientAddress = "Invalid address format";
    }

    // Validate amount
    if (!formData.amountString.trim()) {
      newErrors.amountString = "Amount is required";
    } else {
      const amount = parseFloat(formData.amountString);
      const spendableBalance = parseFloat(wallet.spendableBalanceString);

      if (isNaN(amount) || amount <= 0) {
        newErrors.amountString = "Amount must be a positive number";
      } else if (amount > spendableBalance) {
        newErrors.amountString = `Amount exceeds spendable balance (${formatCurrency(
          wallet.spendableBalanceString,
          wallet.coin
        )})`;
      }
    }

    // Validate memo (optional, but check length if provided)
    if (formData.memo && formData.memo.length > 200) {
      newErrors.memo = "Memo must be 200 characters or less";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    setErrors({});

    try {
      await onSubmit(formData);
    } catch (error) {
      setErrors({
        general:
          error instanceof Error ? error.message : "Failed to create transfer",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleInputChange = (field: keyof TransferFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error for this field when user starts typing
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  const getTransferTypeDescription = () => {
    switch (wallet.walletType) {
      case "custodial":
        return "Instant transfer with automated approval for custodial wallet operations";
      case "hot":
        return "Fast transfer for operational use, may require minimal approval";
      case "warm":
        return "Semi-automated transfer with risk assessment and optional manual review";
      case "cold":
        return "High-security transfer with manual approval process and longer processing time";
      default:
        return "";
    }
  };

  const getEstimatedProcessingTime = () => {
    switch (wallet.walletType) {
      case "custodial":
        return "1-5 minutes";
      case "hot":
        return "5-15 minutes";
      case "warm":
        return "15 minutes - 2 hours";
      case "cold":
        return "1-72 hours";
      default:
        return "Unknown";
    }
  };

  const requiresAdditionalInfo = () => {
    return wallet.walletType === "warm" || wallet.walletType === "cold";
  };

  const isHighValueTransfer = () => {
    const amount = parseFloat(formData.amountString || "0");
    const threshold = wallet.walletType === "cold" ? 1.0 : 10.0; // Different thresholds
    return amount >= threshold;
  };

  return (
    <div className="max-w-2xl mx-auto p-6">
      <Card>
        <CardHeader>
          <CardTitle>Create Transfer</CardTitle>
          <CardDescription>
            Send {wallet.coin.toUpperCase()} from {wallet.label}
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Wallet Information */}
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-medium">From Wallet</h3>
                <Badge
                  variant={
                    wallet.walletType === "cold" ? "secondary" : "default"
                  }
                >
                  {wallet.walletType === "custodial"
                    ? "Warm"
                    : wallet.walletType}{" "}
                  Wallet
                </Badge>
              </div>

              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-gray-500">Wallet:</span>
                  <div className="font-medium">{wallet.label}</div>
                  <div className="text-gray-400">
                    {truncateAddress(wallet.bitgoWalletId)}
                  </div>
                </div>
                <div>
                  <span className="text-gray-500">Spendable Balance:</span>
                  <div className="font-medium text-green-600">
                    {formatCurrency(wallet.spendableBalanceString, wallet.coin)}
                  </div>
                </div>
              </div>

              <div className="mt-3 text-sm text-gray-600">
                <p>{getTransferTypeDescription()}</p>
                <p className="mt-1">
                  <strong>Estimated processing:</strong>{" "}
                  {getEstimatedProcessingTime()}
                </p>
              </div>
            </div>

            {/* General Error */}
            {errors.general && (
              <div className="bg-red-50 border border-red-200 rounded-md p-3">
                <p className="text-red-700 text-sm">{errors.general}</p>
              </div>
            )}

            {/* Recipient Address */}
            <div className="space-y-2">
              <label
                htmlFor="recipientAddress"
                className="block text-sm font-medium text-gray-700"
              >
                Recipient Address <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                id="recipientAddress"
                value={formData.recipientAddress}
                onChange={(e) =>
                  handleInputChange("recipientAddress", e.target.value)
                }
                placeholder={`Enter ${wallet.coin.toUpperCase()} address`}
                className={`w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                  errors.recipientAddress ? "border-red-300" : "border-gray-300"
                }`}
                disabled={isSubmitting}
              />
              {errors.recipientAddress && (
                <p className="text-red-600 text-sm">
                  {errors.recipientAddress}
                </p>
              )}
            </div>

            {/* Amount */}
            <div className="space-y-2">
              <label
                htmlFor="amountString"
                className="block text-sm font-medium text-gray-700"
              >
                Amount <span className="text-red-500">*</span>
              </label>
              <div className="relative">
                <input
                  type="number"
                  id="amountString"
                  value={formData.amountString}
                  onChange={(e) =>
                    handleInputChange("amountString", e.target.value)
                  }
                  placeholder="0.00000000"
                  step="any"
                  min="0"
                  className={`w-full px-3 py-2 pr-16 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    errors.amountString ? "border-red-300" : "border-gray-300"
                  }`}
                  disabled={isSubmitting}
                />
                <div className="absolute inset-y-0 right-0 flex items-center pr-3">
                  <span className="text-gray-500 font-medium">
                    {wallet.coin.toUpperCase()}
                  </span>
                </div>
              </div>
              {errors.amountString && (
                <p className="text-red-600 text-sm">{errors.amountString}</p>
              )}
              <p className="text-sm text-gray-500">
                Available:{" "}
                {formatCurrency(wallet.spendableBalanceString, wallet.coin)}
              </p>
            </div>

            {/* Advanced Options Toggle */}
            <div>
              <button
                type="button"
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="flex items-center text-sm text-blue-600 hover:text-blue-800"
              >
                <span>{showAdvanced ? "Hide" : "Show"} Advanced Options</span>
                <svg
                  className={`ml-1 h-4 w-4 transition-transform ${
                    showAdvanced ? "rotate-180" : ""
                  }`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 9l-7 7-7-7"
                  />
                </svg>
              </button>
            </div>

            {/* Advanced Options */}
            {showAdvanced && (
              <div className="space-y-4 border-t pt-4">
                <div className="space-y-2">
                  <label
                    htmlFor="memo"
                    className="block text-sm font-medium text-gray-700"
                  >
                    Memo (Optional)
                  </label>
                  <textarea
                    id="memo"
                    value={formData.memo}
                    onChange={(e) => handleInputChange("memo", e.target.value)}
                    placeholder="Add a note for this transfer"
                    rows={3}
                    maxLength={200}
                    className={`w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                      errors.memo ? "border-red-300" : "border-gray-300"
                    }`}
                    disabled={isSubmitting}
                  />
                  {errors.memo && (
                    <p className="text-red-600 text-sm">{errors.memo}</p>
                  )}
                  <p className="text-sm text-gray-500">
                    {formData.memo?.length || 0}/200 characters
                  </p>
                </div>

                {/* Warm/Cold Wallet Additional Fields */}
                {requiresAdditionalInfo() && (
                  <>
                    <div className="space-y-2">
                      <label
                        htmlFor="businessPurpose"
                        className="block text-sm font-medium text-gray-700"
                      >
                        Business Purpose{" "}
                        {wallet.walletType === "cold" ||
                        isHighValueTransfer() ? (
                          <span className="text-red-500">*</span>
                        ) : (
                          ""
                        )}
                      </label>
                      <input
                        type="text"
                        id="businessPurpose"
                        value={formData.businessPurpose}
                        onChange={(e) =>
                          handleInputChange("businessPurpose", e.target.value)
                        }
                        placeholder="e.g., Customer withdrawal, Exchange rebalancing"
                        className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                        disabled={isSubmitting}
                      />
                      <p className="text-sm text-gray-500">
                        Explain the business purpose for this transfer
                      </p>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <label
                          htmlFor="requestorName"
                          className="block text-sm font-medium text-gray-700"
                        >
                          Requestor Name <span className="text-red-500">*</span>
                        </label>
                        <input
                          type="text"
                          id="requestorName"
                          value={formData.requestorName}
                          onChange={(e) =>
                            handleInputChange("requestorName", e.target.value)
                          }
                          placeholder="Full name"
                          className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                          disabled={isSubmitting}
                        />
                      </div>

                      <div className="space-y-2">
                        <label
                          htmlFor="requestorEmail"
                          className="block text-sm font-medium text-gray-700"
                        >
                          Requestor Email{" "}
                          <span className="text-red-500">*</span>
                        </label>
                        <input
                          type="email"
                          id="requestorEmail"
                          value={formData.requestorEmail}
                          onChange={(e) =>
                            handleInputChange("requestorEmail", e.target.value)
                          }
                          placeholder="email@company.com"
                          className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                          disabled={isSubmitting}
                        />
                      </div>
                    </div>

                    <div className="space-y-2">
                      <label
                        htmlFor="urgencyLevel"
                        className="block text-sm font-medium text-gray-700"
                      >
                        Urgency Level
                      </label>
                      <select
                        id="urgencyLevel"
                        value={formData.urgencyLevel}
                        onChange={(e) =>
                          handleInputChange("urgencyLevel", e.target.value)
                        }
                        className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                        disabled={isSubmitting}
                      >
                        <option value="low">Low - Standard processing</option>
                        <option value="normal">
                          Normal - Regular priority
                        </option>
                        <option value="high">
                          High - Expedited processing
                        </option>
                        <option value="critical">
                          Critical - Urgent processing
                        </option>
                      </select>
                    </div>

                    {/* Auto-process option for warm wallets */}
                    {wallet.walletType === "warm" && (
                      <div className="space-y-2">
                        <div className="flex items-center">
                          <input
                            type="checkbox"
                            id="autoProcess"
                            checked={formData.autoProcess}
                            onChange={(e) =>
                              setFormData((prev) => ({
                                ...prev,
                                autoProcess: e.target.checked,
                              }))
                            }
                            className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                            disabled={isSubmitting}
                          />
                          <label
                            htmlFor="autoProcess"
                            className="ml-2 text-sm text-gray-700"
                          >
                            Enable automatic processing (if eligible)
                          </label>
                        </div>
                        <p className="text-sm text-gray-500">
                          Small transfers with low risk scores may be processed
                          automatically
                        </p>
                      </div>
                    )}
                  </>
                )}
              </div>
            )}

            {/* Warning for Cold Wallets */}
            {wallet.walletType === "cold" && (
              <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <svg
                      className="h-5 w-5 text-yellow-400"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </div>
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-yellow-800">
                      Cold Wallet Transfer
                    </h3>
                    <div className="mt-1 text-sm text-yellow-700">
                      <p>
                        This transfer requires manual approval and may take up
                        to 72 hours to process. You will be notified when
                        approvals are needed.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Info for Warm Wallets */}
            {wallet.walletType === "warm" && (
              <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <svg
                      className="h-5 w-5 text-blue-400"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </div>
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-blue-800">
                      Warm Wallet Transfer
                    </h3>
                    <div className="mt-1 text-sm text-blue-700">
                      <p>
                        This transfer includes risk assessment and may be
                        processed automatically for small amounts. Larger
                        transfers may require approval.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex justify-end space-x-3 pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={onCancel}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? (
                  <div className="flex items-center">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    Creating...
                  </div>
                ) : (
                  "Create Transfer"
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
