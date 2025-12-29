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

interface CreateWalletFormProps {
  onSubmit: (walletData: CreateWalletFormData) => Promise<void>;
  onCancel: () => void;
}

export interface CreateWalletFormData {
  bitgoWalletId: string;
  label: string;
  coin: string;
  walletType: "custodial" | "hot" | "warm" | "cold";
  multisigType?: string;
  threshold?: number;
  tags: string[];
  metadata?: Record<string, any>;
}

interface FormErrors {
  bitgoWalletId?: string;
  label?: string;
  coin?: string;
  walletType?: string;
  threshold?: string;
  general?: string;
}

const SUPPORTED_COINS = [
  { value: "btc", label: "Bitcoin (BTC)" },
  { value: "eth", label: "Ethereum (ETH)" },
  { value: "usdc", label: "USD Coin (USDC)" },
  { value: "usdt", label: "Tether (USDT)" },
  { value: "ltc", label: "Litecoin (LTC)" },
];

const WALLET_TYPES = [
  {
    value: "custodial" as const,
    label: "Custodial",
    description: "Fully managed by BitGo with instant access",
  },
  {
    value: "hot" as const,
    label: "Hot",
    description: "Online storage for frequent transactions",
  },
  {
    value: "warm" as const,
    label: "Warm",
    description: "Semi-automated with risk assessment",
  },
  {
    value: "cold" as const,
    label: "Cold",
    description: "Offline storage for maximum security",
  },
];

const MULTISIG_TYPES = [
  { value: "onchain", label: "On-chain Multisig" },
  { value: "tss", label: "Threshold Signature Scheme (TSS)" },
  { value: "ecdsa", label: "ECDSA Multisig" },
];

export function CreateWalletForm({
  onSubmit,
  onCancel,
}: CreateWalletFormProps) {
  const [formData, setFormData] = useState<CreateWalletFormData>({
    bitgoWalletId: "",
    label: "",
    coin: "btc",
    walletType: "warm",
    tags: [],
  });

  const [errors, setErrors] = useState<FormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [tagInput, setTagInput] = useState("");

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    // Validate BitGo Wallet ID
    if (!formData.bitgoWalletId.trim()) {
      newErrors.bitgoWalletId = "BitGo Wallet ID is required";
    } else if (!/^[a-fA-F0-9]{24}$/.test(formData.bitgoWalletId)) {
      newErrors.bitgoWalletId =
        "BitGo Wallet ID must be a 24-character hexadecimal string";
    }

    // Validate label
    if (!formData.label.trim()) {
      newErrors.label = "Wallet label is required";
    } else if (formData.label.length > 100) {
      newErrors.label = "Label must be less than 100 characters";
    }

    // Validate threshold for multisig
    if (
      formData.multisigType &&
      (!formData.threshold || formData.threshold < 1 || formData.threshold > 10)
    ) {
      newErrors.threshold =
        "Threshold must be between 1 and 10 for multisig wallets";
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
          error instanceof Error ? error.message : "Failed to create wallet",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleInputChange = (field: keyof CreateWalletFormData, value: any) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error for this field when user starts typing
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  const addTag = () => {
    if (tagInput.trim() && !formData.tags.includes(tagInput.trim())) {
      setFormData((prev) => ({
        ...prev,
        tags: [...prev.tags, tagInput.trim()],
      }));
      setTagInput("");
    }
  };

  const removeTag = (tagToRemove: string) => {
    setFormData((prev) => ({
      ...prev,
      tags: prev.tags.filter((tag) => tag !== tagToRemove),
    }));
  };

  const getWalletTypeDescription = (type: string) => {
    return WALLET_TYPES.find((wt) => wt.value === type)?.description || "";
  };

  return (
    <div className="max-w-2xl mx-auto p-6">
      <Card>
        <CardHeader>
          <CardTitle>Create New Wallet</CardTitle>
          <CardDescription>
            Connect an existing BitGo wallet or create a new one for your
            organization
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* General Error */}
            {errors.general && (
              <div className="bg-red-50 border border-red-200 rounded-md p-3">
                <p className="text-red-700 text-sm">{errors.general}</p>
              </div>
            )}

            {/* BitGo Wallet ID */}
            <div className="space-y-2">
              <label
                htmlFor="bitgoWalletId"
                className="block text-sm font-medium text-gray-700"
              >
                BitGo Wallet ID <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                id="bitgoWalletId"
                value={formData.bitgoWalletId}
                onChange={(e) =>
                  handleInputChange("bitgoWalletId", e.target.value)
                }
                placeholder="64a5b2c8e9f1a2b3c4d5e6f7"
                className={`w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono ${
                  errors.bitgoWalletId ? "border-red-300" : "border-gray-300"
                }`}
                disabled={isSubmitting}
                maxLength={24}
              />
              {errors.bitgoWalletId && (
                <p className="text-red-600 text-sm">{errors.bitgoWalletId}</p>
              )}
              <p className="text-sm text-gray-500">
                The 24-character hexadecimal ID from your BitGo wallet
              </p>
            </div>

            {/* Wallet Label */}
            <div className="space-y-2">
              <label
                htmlFor="label"
                className="block text-sm font-medium text-gray-700"
              >
                Wallet Label <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                id="label"
                value={formData.label}
                onChange={(e) => handleInputChange("label", e.target.value)}
                placeholder="e.g., Main Bitcoin Wallet, Treasury ETH"
                className={`w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                  errors.label ? "border-red-300" : "border-gray-300"
                }`}
                disabled={isSubmitting}
                maxLength={100}
              />
              {errors.label && (
                <p className="text-red-600 text-sm">{errors.label}</p>
              )}
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Coin Selection */}
              <div className="space-y-2">
                <label
                  htmlFor="coin"
                  className="block text-sm font-medium text-gray-700"
                >
                  Cryptocurrency
                </label>
                <select
                  id="coin"
                  value={formData.coin}
                  onChange={(e) => handleInputChange("coin", e.target.value)}
                  className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                  disabled={isSubmitting}
                >
                  {SUPPORTED_COINS.map((coin) => (
                    <option key={coin.value} value={coin.value}>
                      {coin.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Wallet Type */}
              <div className="space-y-2">
                <label
                  htmlFor="walletType"
                  className="block text-sm font-medium text-gray-700"
                >
                  Wallet Type
                </label>
                <select
                  id="walletType"
                  value={formData.walletType}
                  onChange={(e) =>
                    handleInputChange("walletType", e.target.value as any)
                  }
                  className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                  disabled={isSubmitting}
                >
                  {WALLET_TYPES.map((type) => (
                    <option key={type.value} value={type.value}>
                      {type.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Wallet Type Description */}
            <div className="bg-blue-50 border border-blue-200 rounded-md p-3">
              <p className="text-blue-800 text-sm">
                <strong>
                  {
                    WALLET_TYPES.find((t) => t.value === formData.walletType)
                      ?.label
                  }
                  :
                </strong>{" "}
                {getWalletTypeDescription(formData.walletType)}
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
                {/* Multisig Configuration */}
                <div className="space-y-4">
                  <div className="space-y-2">
                    <label
                      htmlFor="multisigType"
                      className="block text-sm font-medium text-gray-700"
                    >
                      Multisig Type (Optional)
                    </label>
                    <select
                      id="multisigType"
                      value={formData.multisigType || ""}
                      onChange={(e) =>
                        handleInputChange(
                          "multisigType",
                          e.target.value || undefined
                        )
                      }
                      className="w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                      disabled={isSubmitting}
                    >
                      <option value="">None (Single-signature)</option>
                      {MULTISIG_TYPES.map((type) => (
                        <option key={type.value} value={type.value}>
                          {type.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  {formData.multisigType && (
                    <div className="space-y-2">
                      <label
                        htmlFor="threshold"
                        className="block text-sm font-medium text-gray-700"
                      >
                        Signature Threshold{" "}
                        <span className="text-red-500">*</span>
                      </label>
                      <input
                        type="number"
                        id="threshold"
                        value={formData.threshold || ""}
                        onChange={(e) =>
                          handleInputChange(
                            "threshold",
                            parseInt(e.target.value) || undefined
                          )
                        }
                        placeholder="2"
                        min="1"
                        max="10"
                        className={`w-full px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                          errors.threshold
                            ? "border-red-300"
                            : "border-gray-300"
                        }`}
                        disabled={isSubmitting}
                      />
                      {errors.threshold && (
                        <p className="text-red-600 text-sm">
                          {errors.threshold}
                        </p>
                      )}
                      <p className="text-sm text-gray-500">
                        Number of signatures required to authorize transactions
                      </p>
                    </div>
                  )}
                </div>

                {/* Tags */}
                <div className="space-y-2">
                  <label className="block text-sm font-medium text-gray-700">
                    Tags (Optional)
                  </label>
                  <div className="flex space-x-2">
                    <input
                      type="text"
                      value={tagInput}
                      onChange={(e) => setTagInput(e.target.value)}
                      onKeyPress={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          addTag();
                        }
                      }}
                      placeholder="Add tag..."
                      className="flex-1 px-3 py-2 border rounded-md text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 border-gray-300"
                      disabled={isSubmitting}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={addTag}
                      disabled={!tagInput.trim() || isSubmitting}
                    >
                      Add
                    </Button>
                  </div>
                  {formData.tags.length > 0 && (
                    <div className="flex flex-wrap gap-2 mt-2">
                      {formData.tags.map((tag) => (
                        <Badge
                          key={tag}
                          variant="secondary"
                          className="flex items-center gap-1"
                        >
                          {tag}
                          <button
                            type="button"
                            onClick={() => removeTag(tag)}
                            className="ml-1 text-gray-500 hover:text-gray-700"
                            disabled={isSubmitting}
                          >
                            ×
                          </button>
                        </Badge>
                      ))}
                    </div>
                  )}
                  <p className="text-sm text-gray-500">
                    Add tags to organize and categorize your wallets
                  </p>
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
                  "Create Wallet"
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
