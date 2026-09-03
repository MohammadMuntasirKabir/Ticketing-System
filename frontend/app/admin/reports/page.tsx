"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Loader2 } from "lucide-react";
import { adminApi, type LoadFactorReport, type RevenueReport, type PeakTimesReport } from "@/lib/api-client";
import { getToken } from "@/lib/auth";

export default function AdminReportsPage() {
  const [loadFactor, setLoadFactor] = useState<LoadFactorReport[]>([]);
  const [revenue, setRevenue] = useState<RevenueReport[]>([]);
  const [peakTimes, setPeakTimes] = useState<PeakTimesReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const token = getToken();
        const [lf, rev, pt] = await Promise.all([
          adminApi.reports.loadFactor(token),
          adminApi.reports.revenue(token),
          adminApi.reports.peakTimes(token),
        ]);
        setLoadFactor(lf);
        setRevenue(rev);
        setPeakTimes(pt);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load reports");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (loading) {
    return <div className="flex items-center justify-center gap-2 py-20 text-gray-500"><Loader2 className="h-5 w-5 animate-spin" /> Loading reports…</div>;
  }

  if (error) {
    return <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">{error}</div>;
  }

  const totalSold = loadFactor.reduce((s, r) => s + r.sold, 0);
  const totalCap = loadFactor.reduce((s, r) => s + r.capacity, 0);
  const totalRevenue = revenue.reduce((s, r) => s + r.revenue, 0);
  const busiestHour = peakTimes.reduce<PeakTimesReport | null>((best, r) => (best === null || r.bookings > best.bookings ? r : best), null);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold tracking-tight">Reports</h1><p className="text-sm text-gray-500 dark:text-gray-400">Operational analytics</p></div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader><CardTitle className="text-sm text-gray-500">Overall Load Factor</CardTitle></CardHeader>
          <CardContent className="text-3xl font-bold">{totalCap > 0 ? ((totalSold / totalCap) * 100).toFixed(1) : "0.0"}%</CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm text-gray-500">Total Revenue</CardTitle></CardHeader>
          <CardContent className="text-3xl font-bold">${totalRevenue.toFixed(2)}</CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm text-gray-500">Busiest Hour</CardTitle></CardHeader>
          <CardContent className="text-3xl font-bold">{busiestHour ? `${busiestHour.hour}:00` : "—"}</CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader><CardTitle>Load Factor by Flight</CardTitle></CardHeader>
        <CardContent>
          <div className="space-y-3">
            {loadFactor.map(r => (
              <div key={r.flight_instance_id} className="flex items-center gap-3">
                <div className="w-28 text-sm font-medium">{r.flight_number}</div>
                <div className="flex-1">
                  <div className="h-2 rounded-full bg-gray-200 dark:bg-gray-700">
                    <div className="h-2 rounded-full bg-brand-gradient" style={{ width: `${Math.min(r.load_factor * 100, 100)}%` }} />
                  </div>
                </div>
                <div className="w-20 text-right text-sm">{r.sold}/{r.capacity} ({Math.round(r.load_factor * 100)}%)</div>
              </div>
            ))}
            {loadFactor.length === 0 && <p className="text-sm text-gray-500">No flight data.</p>}
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Revenue by Flight</CardTitle></CardHeader>
          <CardContent>
            <div className="space-y-2">
              {revenue.map(r => (
                <div key={r.flight_instance_id} className="flex items-center justify-between text-sm">
                  <span className="font-medium">{r.flight_number}</span>
                  <span>${r.revenue.toFixed(2)} · {r.bookings} bookings</span>
                </div>
              ))}
              {revenue.length === 0 && <p className="text-sm text-gray-500">No revenue data.</p>}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Bookings by Hour</CardTitle></CardHeader>
          <CardContent>
            <div className="space-y-2">
              {peakTimes.map(r => (
                <div key={r.hour} className="flex items-center justify-between text-sm">
                  <span className="font-medium">{String(r.hour).padStart(2, "0")}:00</span>
                  <span>{r.bookings} bookings</span>
                </div>
              ))}
              {peakTimes.length === 0 && <p className="text-sm text-gray-500">No data.</p>}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
