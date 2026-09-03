"use client";

import { useEffect, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Card, CardContent } from "@/components/ui/card";
import { Loader2 } from "lucide-react";
import { adminApi, type AdminBookingRow } from "@/lib/api-client";
import { getToken } from "@/lib/auth";

export default function AdminBookingsPage() {
  const [bookings, setBookings] = useState<AdminBookingRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const res = await adminApi.bookings.list(getToken());
        setBookings(res.data);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load bookings");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold tracking-tight">Bookings</h1><p className="text-sm text-gray-500 dark:text-gray-400">All booking records across flights</p></div>
      {error && <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">{error}</div>}
      <Card><CardContent className="p-0">
        {loading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-gray-500"><Loader2 className="h-5 w-5 animate-spin" /> Loading bookings…</div>
        ) : (
          <Table>
            <TableHeader><TableRow><TableHead>ID</TableHead><TableHead>PNR ID</TableHead><TableHead>Flight</TableHead><TableHead>Status</TableHead><TableHead>Total</TableHead></TableRow></TableHeader>
            <TableBody>
              {bookings.map(b => (
                <TableRow key={b.id}>
                  <TableCell className="font-semibold">#{b.id}</TableCell>
                  <TableCell>{b.pnr_id}</TableCell>
                  <TableCell>{b.flight_instance_id}</TableCell>
                  <TableCell>{b.status}</TableCell>
                  <TableCell>${b.total_price.toFixed(2)} {b.currency}</TableCell>
                </TableRow>
              ))}
              {bookings.length === 0 && (
                <TableRow><TableCell colSpan={5} className="py-8 text-center text-gray-500">No bookings found.</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </CardContent></Card>
    </div>
  );
}
