"use client";

import React from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatCurrency, formatTimeAgo, truncateAddress } from "@/lib/utils";

export interface TransferStatus {
  status: string;
  timestamp: string;
  description: string;
  isCompleted: boolean;
  isCurrent: boolean;
  details?: string;
}

export interface Transfer {
  id: string;
  walletId: string;
  walletLabel: string;
  recipientAddress: string;
  amountString: string;
  coin: string;
  transferType: "custodial" | "hot" | "cold";
  status: string;
  bitgoTransferId?: string;
  transactionHash?: string;
  requiredApprovals: number;
  receivedApprovals: number;
  memo?: string;
  fee?: string;
  feeRate?: string;
  submittedAt?: string;
  approvedAt?: string;
  completedAt?: string;
  failedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ApprovalInfo {
  id: string;
  requiredApprovals: number;
  receivedApprovals: number;
  pendingApprovals: number;
  approvers: Array<{
    userId: string;
    username: string;
    state: string;
    approvalDate?: string;
  }>;
  timeRemaining: string;
  isExpired: boolean;
  canUserApprove: boolean;
}

interface TransferDetailProps {
  transfer: Transfer;
  approvalInfo?: ApprovalInfo;
  onApprove?: () => Promise<void>;
  onReject?: () => Promise<void>;
  onSubmit?: () => Promise<void>;
  onCancel?: () => Promise<void>;
}

export function TransferDetail({
  transfer,
  approvalInfo,
  onApprove,
  onReject,
  onSubmit,
  onCancel,
}: TransferDetailProps) {
  const getStatusVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case "completed":
      case "confirmed":
        return "success";
      case "failed":
      case "rejected":
        return "destructive";
      case "pending_approval":
        return "warning";
      case "broadcast":
      case "submitted":
        return "default";
      default:
        return "outline";
    }
  };

  const getTransferTimeline = (): TransferStatus[] => {
    const timeline: TransferStatus[] = [
      {
        status: "created",
        timestamp: transfer.createdAt,
        description: "Transfer created",
        isCompleted: true,
        isCurrent: false,
        details: `Transfer request created for ${formatCurrency(
          transfer.amountString,
          transfer.coin
        )}`,
      },
    ];

    if (
      transfer.status === "pending_approval" ||
      transfer.receivedApprovals > 0
    ) {
      timeline.push({
        status: "pending_approval",
        timestamp: transfer.createdAt,
        description: "Awaiting approvals",
        isCompleted: transfer.status !== "pending_approval",
        isCurrent: transfer.status === "pending_approval",
        details: approvalInfo
          ? `${approvalInfo.receivedApprovals}/${approvalInfo.requiredApprovals} approvals received`
          : `${transfer.receivedApprovals}/${transfer.requiredApprovals} approvals received`,
      });
    }

    if (
      transfer.approvedAt ||
      ["approved", "signed", "broadcast", "completed"].includes(transfer.status)
    ) {
      timeline.push({
        status: "approved",
        timestamp: transfer.approvedAt || transfer.updatedAt,
        description: "Transfer approved",
        isCompleted: true,
        isCurrent: transfer.status === "approved",
        details: "All required approvals received",
      });
    }

    if (
      transfer.submittedAt ||
      ["signed", "broadcast", "completed"].includes(transfer.status)
    ) {
      timeline.push({
        status: "submitted",
        timestamp: transfer.submittedAt || transfer.updatedAt,
        description: "Submitted to blockchain",
        isCompleted: ["broadcast", "completed"].includes(transfer.status),
        isCurrent:
          transfer.status === "signed" || transfer.status === "submitted",
        details: "Transaction signed and submitted to the blockchain network",
      });
    }

    if (transfer.status === "broadcast") {
      timeline.push({
        status: "broadcast",
        timestamp: transfer.updatedAt,
        description: "Transaction broadcast",
        isCompleted: false,
        isCurrent: true,
        details: "Transaction broadcast to network, awaiting confirmations",
      });
    }

    if (transfer.completedAt || transfer.status === "completed") {
      timeline.push({
        status: "completed",
        timestamp: transfer.completedAt || transfer.updatedAt,
        description: "Transfer completed",
        isCompleted: true,
        isCurrent: transfer.status === "completed",
        details: "Transaction confirmed on blockchain",
      });
    }

    if (transfer.failedAt || transfer.status === "failed") {
      timeline.push({
        status: "failed",
        timestamp: transfer.failedAt || transfer.updatedAt,
        description: "Transfer failed",
        isCompleted: true,
        isCurrent: transfer.status === "failed",
        details: "Transfer failed to complete",
      });
    }

    return timeline;
  };

  const timeline = getTransferTimeline();

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Transfer Details</h1>
          <p className="text-gray-600 mt-1">Transfer ID: {transfer.id}</p>
        </div>
        <Badge
          variant={getStatusVariant(transfer.status)}
          className="text-lg px-4 py-2"
        >
          {transfer.status.replace("_", " ").toUpperCase()}
        </Badge>
      </div>

      {/* Transfer Summary */}
      <Card>
        <CardHeader>
          <CardTitle>Transfer Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-500">
                  Amount
                </label>
                <div className="text-2xl font-bold text-green-600">
                  {formatCurrency(transfer.amountString, transfer.coin)}
                </div>
              </div>

              <div>
                <label className="text-sm font-medium text-gray-500">
                  From Wallet
                </label>
                <div className="font-medium">{transfer.walletLabel}</div>
                <div className="text-sm text-gray-500">
                  {transfer.transferType === "custodial"
                    ? "Warm"
                    : transfer.transferType}{" "}
                  Wallet
                </div>
              </div>

              <div>
                <label className="text-sm font-medium text-gray-500">
                  Recipient
                </label>
                <div className="font-mono text-sm break-all">
                  {transfer.recipientAddress}
                </div>
              </div>
            </div>

            <div className="space-y-4">
              {transfer.transactionHash && (
                <div>
                  <label className="text-sm font-medium text-gray-500">
                    Transaction Hash
                  </label>
                  <div className="font-mono text-sm break-all">
                    {transfer.transactionHash}
                  </div>
                  <Button variant="link" className="p-0 h-auto text-blue-600">
                    View on Explorer
                  </Button>
                </div>
              )}

              {transfer.fee && (
                <div>
                  <label className="text-sm font-medium text-gray-500">
                    Network Fee
                  </label>
                  <div className="font-medium">
                    {formatCurrency(transfer.fee, transfer.coin)}
                  </div>
                  {transfer.feeRate && (
                    <div className="text-sm text-gray-500">
                      Rate: {transfer.feeRate} sat/byte
                    </div>
                  )}
                </div>
              )}

              {transfer.memo && (
                <div>
                  <label className="text-sm font-medium text-gray-500">
                    Memo
                  </label>
                  <div className="text-sm bg-gray-50 p-2 rounded">
                    {transfer.memo}
                  </div>
                </div>
              )}

              <div>
                <label className="text-sm font-medium text-gray-500">
                  Created
                </label>
                <div className="text-sm">
                  {formatTimeAgo(new Date(transfer.createdAt))}
                </div>
                <div className="text-xs text-gray-400">
                  {new Date(transfer.createdAt).toLocaleString()}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Pending Approval Info */}
      {approvalInfo && transfer.status === "pending_approval" && (
        <Card className="border-yellow-200 bg-yellow-50">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <span className="text-yellow-600">⏳</span>
              <span>Pending Approvals</span>
            </CardTitle>
            <CardDescription>
              This transfer requires approval before it can be submitted.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-lg font-semibold">
                    {approvalInfo.receivedApprovals} of{" "}
                    {approvalInfo.requiredApprovals} approvals received
                  </div>
                  <div className="text-sm text-gray-600">
                    {approvalInfo.pendingApprovals} approval(s) pending
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-sm text-gray-500">Time remaining</div>
                  <div
                    className={`font-medium ${
                      approvalInfo.isExpired ? "text-red-600" : "text-gray-900"
                    }`}
                  >
                    {approvalInfo.isExpired
                      ? "Expired"
                      : approvalInfo.timeRemaining}
                  </div>
                </div>
              </div>

              {/* Approvers List */}
              <div className="space-y-2">
                <h4 className="font-medium text-gray-900">Approvers</h4>
                <div className="space-y-2">
                  {approvalInfo.approvers.map((approver, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between p-2 bg-white rounded border"
                    >
                      <div>
                        <div className="font-medium">{approver.username}</div>
                        <div className="text-sm text-gray-500">
                          {approver.userId}
                        </div>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Badge
                          variant={
                            approver.state === "approved"
                              ? "success"
                              : "outline"
                          }
                        >
                          {approver.state === "approved"
                            ? "Approved"
                            : "Pending"}
                        </Badge>
                        {approver.approvalDate && (
                          <div className="text-xs text-gray-400">
                            {formatTimeAgo(new Date(approver.approvalDate))}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Action Buttons */}
              {approvalInfo.canUserApprove && !approvalInfo.isExpired && (
                <div className="flex space-x-3">
                  <Button onClick={onApprove} className="flex-1">
                    Approve Transfer
                  </Button>
                  <Button
                    onClick={onReject}
                    variant="destructive"
                    className="flex-1"
                  >
                    Reject Transfer
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Status Timeline */}
      <Card>
        <CardHeader>
          <CardTitle>Transfer Timeline</CardTitle>
          <CardDescription>
            Track the progress of your transfer from creation to completion
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="relative">
            {timeline.map((step, index) => (
              <div
                key={index}
                className="relative flex items-start space-x-3 pb-6"
              >
                {/* Timeline Line */}
                {index < timeline.length - 1 && (
                  <div className="absolute left-4 top-8 h-full w-0.5 bg-gray-200" />
                )}

                {/* Timeline Marker */}
                <div
                  className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                    step.isCompleted
                      ? "bg-green-500 text-white"
                      : step.isCurrent
                      ? "bg-blue-500 text-white"
                      : "bg-gray-200 text-gray-500"
                  }`}
                >
                  {step.isCompleted ? "✓" : index + 1}
                </div>

                {/* Timeline Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2">
                    <h3
                      className={`font-medium ${
                        step.isCurrent ? "text-blue-600" : "text-gray-900"
                      }`}
                    >
                      {step.description}
                    </h3>
                    {step.isCurrent && (
                      <Badge variant="default" className="text-xs">
                        Current
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm text-gray-600 mt-1">{step.details}</p>
                  <p className="text-xs text-gray-400 mt-1">
                    {formatTimeAgo(new Date(step.timestamp))} •{" "}
                    {new Date(step.timestamp).toLocaleString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Action Buttons */}
      {transfer.status === "approved" && onSubmit && (
        <Card className="border-green-200 bg-green-50">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium text-green-800">Ready to Submit</h3>
                <p className="text-sm text-green-700">
                  This transfer has been approved and is ready to be submitted
                  to the blockchain.
                </p>
              </div>
              <Button
                onClick={onSubmit}
                className="bg-green-600 hover:bg-green-700"
              >
                Submit Transfer
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {transfer.status === "draft" && onCancel && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium text-gray-800">Transfer Actions</h3>
                <p className="text-sm text-gray-600">
                  This transfer is still in draft status.
                </p>
              </div>
              <div className="space-x-2">
                <Button variant="outline">Edit Transfer</Button>
                <Button onClick={onCancel} variant="destructive">
                  Cancel Transfer
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
