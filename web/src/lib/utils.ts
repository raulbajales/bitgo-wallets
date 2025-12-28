import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}

export function formatAmount(amount: string, decimals: number = 8): string {
    const num = parseFloat(amount)
    if (isNaN(num)) return '0'
    return num.toLocaleString('en-US', {
        minimumFractionDigits: 0,
        maximumFractionDigits: decimals,
    })
}

export function formatCurrency(amount: string, currency: string): string {
    return `${formatAmount(amount)} ${currency.toUpperCase()}`
}

export function truncateAddress(address: string, start: number = 6, end: number = 4): string {
    if (address.length <= start + end) return address
    return `${address.slice(0, start)}...${address.slice(-end)}`
}

export function formatTimeAgo(date: Date): string {
    const now = new Date()
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000)

    if (diffInSeconds < 60) return 'Just now'
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`
    return `${Math.floor(diffInSeconds / 86400)}d ago`
}