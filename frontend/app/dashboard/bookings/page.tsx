"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2, AlertCircle, Search } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { bookingApi, type BookingDetail } from "@/lib/api-client";

function BookingLookup() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialPnr = searchParams?.get("pnr") || "";
  const [pnr, setPnr] = useState(initialPnr);
  const [detail, setDetail] = useState<BookingDetail | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async (code: string) => {
    if (!code.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const data = await bookingApi.getByPNR(code.trim().toUpperCase());
      setDetail(data);
    } catch (e) {
      setDetail(null);
      setError(e instanceof Error ? e.message : "Booking not found");
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    if (!initialPnr) return;
    let cancelled = false;
    (async () => {
      try {
        const data = await bookingApi.getByPNR(initialPnr.trim().toUpperCase());
        if (!cancelled) setDetail(data);
      } catch (e) {
        if (cancelled) return;
        setDetail(null);
        setError(e instanceof Error ? e.message : "Booking not found");
      }
    })();
    return () => { cancelled = true; };
  }, [initialPnr]);

  const statusTone = (status: string) =>
    status === "CONFIRMED" ? "text-green-600 dark:text-green-400"
    : status === "CANCELLED" ? "text-red-600 dark:text-red-400"
    : "text-amber-600 dark:text-amber-400";

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">My Bookings</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">Look up a booking by its reference (PNR)</p>
      </div>

      <form onSubmit={e => { e.preventDefault(); load(pnr); }} className="flex gap-2">
        <input
          type="text"
          required
          maxLength={6}
          value={pnr}
          onChange={e => setPnr(e.target.value.toUpperCase())}
          placeholder="Enter PNR (e.g. A7KF3L)"
          className="flex h-10 flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
        />
        <Button type="submit" className="btn-gradient" disabled={busy}>
          {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
        </Button>
      </form>

      {error && (
        <Card><CardContent className="p-6 flex items-center gap-3 text-red-600 dark:text-red-400">
          <AlertCircle className="h-5 w-5" /> <span className="text-sm">{error}</span>
        </CardContent></Card>
      )}

      {detail && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>PNR {detail.pnr}</span>
              <span className={`text-sm font-medium ${statusTone(detail.status)}`}>{detail.status}</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div><div className="text-gray-500">Total</div><div className="font-medium">${(detail.total_cents / 100).toFixed(2)} {detail.currency}</div></div>
              <div><div className="text-gray-500">Booked</div><div className="font-medium">{new Date(detail.created_at).toLocaleString()}</div></div>
            </div>
            <div className="space-y-2">
              {detail.bookings.map(b => (
                <div key={b.id} className="flex items-center justify-between rounded-lg border border-gray-200 p-3 text-sm dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <span className="font-semibold">{b.flight_number}</span>
                    <span className="text-gray-500">{b.fare_class}</span>
                    {b.seat && <span className="rounded bg-gray-100 px-1.5 py-0.5 text-xs dark:bg-gray-800">Seat {b.seat}</span>}
                  </div>
                  <span className={statusTone(b.status)}>{b.status}</span>
                </div>
              ))}
            </div>
            <div className="flex gap-3 pt-2">
              <Button variant="outline" className="flex-1" onClick={() => router.push(`/check-in/${detail.pnr}`)}>Check In</Button>
              <Button variant="outline" className="flex-1" onClick={() => router.push(`/boarding-pass/${detail.pnr}`)}>Boarding Pass</Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

export default function DashboardBookingsPage() {
  return (
    <div className="min-h-screen bg-gray-50 py-10 dark:bg-gray-950">
      <div className="container px-4">
        <Suspense fallback={<div className="flex justify-center py-20"><Loader2 className="h-6 w-6 animate-spin text-gray-400" /></div>}>
          <BookingLookup />
        </Suspense>
      </div>
    </div>
  );
}
