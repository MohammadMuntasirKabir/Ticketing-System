"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { SeatMap } from "@/components/seat-map/seat-map";
import { StripePaymentForm } from "@/components/payment/payment-form";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ArrowRight, Plane, Loader2, AlertCircle } from "lucide-react";
import { flightApi, bookingApi, type SearchResult, type SeatMapData, type CreateBookingResponse, type PassengerInput } from "@/lib/api-client";
import { getStripe, stripeConfigured } from "@/lib/stripe";

const emptyPassenger = (): PassengerInput => ({
  title: "Mr",
  first_name: "",
  last_name: "",
  date_of_birth: "",
  nationality: "US",
  passport_number: "",
  passport_expiry: "",
});

const makeIdempotencyKey = (id: number) => `${id}-${Date.now()}`;

const STORE_KEY = "at_pending_booking";

interface StoredBooking {
  pnr: string;
  client_secret: string;
  amount_cents: number;
  currency: string;
}

export default function FlightDetailPage() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const id = Number(params?.id ?? 0);

  const count = Math.min(Math.max(parseInt(searchParams?.get("passengers") || "1", 10) || 1, 1), 9);

  const [flight, setFlight] = useState<SearchResult | null>(null);
  const [seatMap, setSeatMap] = useState<SeatMapData | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [selectedSeats, setSelectedSeats] = useState<string[]>([]);
  const [passengers, setPassengers] = useState<PassengerInput[]>(() => Array.from({ length: count }, emptyPassenger));
  const [step, setStep] = useState<"seats" | "passengers" | "booking">("seats");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [booking, setBooking] = useState<CreateBookingResponse | null>(null);
  const [confirmed, setConfirmed] = useState(false);

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    (async () => {
      try {
        const [detail, sm] = await Promise.all([
          flightApi.getDetail(id),
          flightApi.getSeatMap(id),
        ]);
        if (cancelled) return;
        setFlight(detail);
        setSeatMap(sm);
      } catch (e) {
        if (!cancelled) setLoadError(e instanceof Error ? e.message : "Failed to load flight");
      }
    })();
    return () => { cancelled = true; };
  }, [id]);

  // Recover an in-flight booking after a Stripe redirect method returns.
  useEffect(() => {
    if (!id) return;
    const secret = searchParams?.get("payment_intent_client_secret");
    const pnr = searchParams?.get("pnr");
    const isReturn = searchParams?.get("payment") === "return";
    if (!secret && !(isReturn && pnr)) return;
    let cancelled = false;
    (async () => {
      if (secret) {
        const stripe = await getStripe();
        if (!stripe || cancelled) return;
        const { paymentIntent } = await stripe.retrievePaymentIntent(secret);
        if (cancelled || !paymentIntent) return;
        setBooking({
          pnr: pnr || "",
          pnr_id: 0,
          status: "PENDING",
          payment_intent_id: paymentIntent.id,
          payment_client_secret: secret,
          amount_cents: paymentIntent.amount,
          currency: paymentIntent.currency.toUpperCase(),
          hold_expires_at: "",
        });
        if (paymentIntent.status === "succeeded") {
          sessionStorage.removeItem(STORE_KEY);
          setConfirmed(true);
        }
        return;
      }
      const raw = sessionStorage.getItem(STORE_KEY);
      if (!raw) return;
      try {
        const stored = JSON.parse(raw) as StoredBooking;
        if (stored.pnr !== pnr) return;
        setBooking({
          pnr: stored.pnr,
          pnr_id: 0,
          status: "PENDING",
          payment_intent_id: "",
          payment_client_secret: stored.client_secret,
          amount_cents: stored.amount_cents,
          currency: stored.currency,
          hold_expires_at: "",
        });
        const res = await bookingApi.confirm(stored.pnr);
        if (!cancelled && res.status === "CONFIRMED") {
          sessionStorage.removeItem(STORE_KEY);
          setConfirmed(true);
        }
      } catch {
        // ignore recovery failures; the manual confirm button remains available
      }
    })();
    return () => { cancelled = true; };
  }, [id, searchParams]);

  const cheapest = flight ? [...flight.fares].sort((a, b) => a.total - b.total)[0] : null;
  const fareCode = cheapest?.fare_class_code || "YSAVER";

  const handleContinueToPassengers = () => {
    if (selectedSeats.length > 0) setStep("passengers");
  };

  const handleBook = async () => {
    const allFilled = passengers.every(p => p.first_name && p.last_name && p.passport_number);
    if (!allFilled) {
      setError("Please fill in all passenger details.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const quote = await bookingApi.getPrice({
        flight_instance_id: id,
        fare_class_code: fareCode,
        passengers: count,
      });
      const created = await bookingApi.create({
        quote_id: quote.quote_id,
        idempotency_key: makeIdempotencyKey(id),
        passengers: passengers.map(p => ({ ...p, date_of_birth: p.date_of_birth || "1990-01-01", passport_expiry: p.passport_expiry || "2030-01-01" })),
      });
      setBooking(created);
      setStep("booking");
      sessionStorage.setItem(STORE_KEY, JSON.stringify({
        pnr: created.pnr,
        client_secret: created.payment_client_secret,
        amount_cents: created.amount_cents,
        currency: created.currency,
      } satisfies StoredBooking));
      if (selectedSeats.length > 0) {
        bookingApi.selectSeats(created.pnr, selectedSeats[0]).catch(() => {});
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Booking failed");
    } finally {
      setBusy(false);
    }
  };

  const handleConfirm = async () => {
    if (!booking) return;
    setBusy(true);
    setError(null);
    try {
      await bookingApi.confirm(booking.pnr);
      sessionStorage.removeItem(STORE_KEY);
      setConfirmed(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Payment confirmation failed");
    } finally {
      setBusy(false);
    }
  };

  if (loadError) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="text-center">
          <AlertCircle className="mx-auto mb-4 h-12 w-12 text-red-400" />
          <p className="mb-4 text-gray-600 dark:text-gray-400">{loadError}</p>
          <Button variant="outline" onClick={() => router.back()}>Go back</Button>
        </div>
      </div>
    );
  }

  if (!flight || !seatMap) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  const departure = new Date(flight.departure);
  const arrival = new Date(flight.arrival);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <div className="container mx-auto px-4 py-8">
        <div className="mb-4 flex items-center gap-2">
          <button onClick={() => router.back()} className="text-sm text-gray-500 hover:text-gray-900">&larr; Back to results</button>
        </div>

        <Card className="mb-6">
          <CardContent className="p-4 flex items-center justify-between flex-wrap gap-4">
            <div className="flex items-center gap-4">
              <div className="text-2xl font-bold">{flight.flight_number}</div>
              <div className="text-sm text-gray-500">{flight.airline.name} · {flight.aircraft}</div>
            </div>
            <div className="flex items-center gap-4 text-sm">
              <span className="font-semibold">{flight.origin.code}</span>
              <span>{departure.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })} → {arrival.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span>
              <span className="font-semibold">{flight.destination.code}</span>
              <span className="text-gray-400">({Math.floor(flight.duration_minutes / 60)}h {flight.duration_minutes % 60}m)</span>
            </div>
            <div className="text-right">
              <div className="text-lg font-bold">${(cheapest?.total || 0).toFixed(2)}</div>
              <div className="text-xs text-gray-500">per passenger</div>
            </div>
          </CardContent>
        </Card>

        {error && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">{error}</div>
        )}

        {step === "seats" && (
          <div className="grid gap-6 lg:grid-cols-[1fr_300px]">
            <Card>
              <CardHeader><CardTitle>Select Your Seat</CardTitle><CardDescription>Choose up to {count} seat{count > 1 ? "s" : ""} for your journey</CardDescription></CardHeader>
              <CardContent><SeatMap data={seatMap} selectedSeats={selectedSeats} onSelect={setSelectedSeats} maxSeats={count} /></CardContent>
            </Card>
            <div className="space-y-4">
              <Card><CardContent className="p-4">
                <div className="text-sm font-medium mb-2">Booking Summary</div>
                <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <div className="flex justify-between"><span>Flight</span><span>{flight.flight_number}</span></div>
                  <div className="flex justify-between"><span>Route</span><span>{flight.origin.code} → {flight.destination.code}</span></div>
                  <div className="flex justify-between"><span>Cabin</span><span>{cheapest?.cabin || "—"}</span></div>
                  <div className="flex justify-between"><span>Passengers</span><span>{count}</span></div>
                  <div className="flex justify-between"><span>Seats</span><span>{selectedSeats.join(", ") || "None"}</span></div>
                  <hr className="border-gray-200 dark:border-gray-700" />
                  <div className="flex justify-between font-semibold text-gray-900 dark:text-gray-100"><span>Total</span><span>${((cheapest?.total || 0) * count).toFixed(2)}</span></div>
                </div>
              </CardContent></Card>
              <Button className="btn-gradient w-full" disabled={selectedSeats.length === 0} onClick={handleContinueToPassengers}>Continue <ArrowRight className="ml-1 h-4 w-4" /></Button>
            </div>
          </div>
        )}

        {step === "passengers" && (
          <div className="grid gap-6 lg:grid-cols-[1fr_300px]">
            <Card>
              <CardHeader><CardTitle>Passenger Details</CardTitle><CardDescription>Enter details for each traveler</CardDescription></CardHeader>
              <CardContent>
                <div className="space-y-6">
                  {passengers.map((p, i) => (
                    <div key={i} className="grid gap-4 sm:grid-cols-2 border-t border-gray-200 dark:border-gray-700 pt-4">
                      <div className="text-sm font-medium text-gray-500">Passenger {i + 1}</div>
                      <select value={p.title} onChange={e => { const np = [...passengers]; np[i] = { ...np[i], title: e.target.value }; setPassengers(np); }} className="rounded-md border border-input bg-background px-3 py-2 text-sm">
                        <option>Mr</option><option>Mrs</option><option>Ms</option><option>Dr</option>
                      </select>
                      <input placeholder="First name" value={p.first_name} onChange={e => { const np = [...passengers]; np[i] = { ...np[i], first_name: e.target.value }; setPassengers(np); }} className="rounded-md border border-input bg-background px-3 py-2 text-sm" />
                      <input placeholder="Last name" value={p.last_name} onChange={e => { const np = [...passengers]; np[i] = { ...np[i], last_name: e.target.value }; setPassengers(np); }} className="rounded-md border border-input bg-background px-3 py-2 text-sm" />
                      <input type="date" value={p.date_of_birth} onChange={e => { const np = [...passengers]; np[i] = { ...np[i], date_of_birth: e.target.value }; setPassengers(np); }} className="rounded-md border border-input bg-background px-3 py-2 text-sm" />
                      <input placeholder="Passport number" value={p.passport_number} onChange={e => { const np = [...passengers]; np[i] = { ...np[i], passport_number: e.target.value }; setPassengers(np); }} className="rounded-md border border-input bg-background px-3 py-2 text-sm sm:col-span-2" />
                      <input type="date" value={p.passport_expiry} onChange={e => { const np = [...passengers]; np[i] = { ...np[i], passport_expiry: e.target.value }; setPassengers(np); }} className="rounded-md border border-input bg-background px-3 py-2 text-sm" />
                    </div>
                  ))}
                </div>
              </CardContent>
              <CardContent>
                <Button className="btn-gradient w-full" onClick={handleBook} disabled={busy}>
                  {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "Book Now"} <ArrowRight className="ml-1 h-4 w-4" />
                </Button>
              </CardContent>
            </Card>
            <div className="space-y-4">
              <Card><CardContent className="p-4">
                <div className="text-sm font-medium mb-2">Booking Summary</div>
                <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <div className="flex justify-between"><span>Flight</span><span>{flight.flight_number}</span></div>
                  <div className="flex justify-between"><span>Route</span><span>{flight.origin.code} → {flight.destination.code}</span></div>
                  <div className="flex justify-between"><span>Passengers</span><span>{count}</span></div>
                  <div className="flex justify-between"><span>Seats</span><span>{selectedSeats.join(", ") || "None"}</span></div>
                  <hr className="border-gray-200 dark:border-gray-700" />
                  <div className="flex justify-between font-semibold text-gray-900 dark:text-gray-100"><span>Total</span><span>${((cheapest?.total || 0) * count).toFixed(2)}</span></div>
                </div>
              </CardContent></Card>
            </div>
          </div>
        )}

        {step === "booking" && booking && (
          <div className="mx-auto max-w-lg text-center py-8">
            <div className={`mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full ${confirmed ? "bg-green-100 dark:bg-green-900/30" : "bg-blue-100 dark:bg-blue-900/30"}`}>
              {confirmed ? <Plane className="h-8 w-8 text-green-600 dark:text-green-400" /> : <Loader2 className="h-8 w-8 animate-spin text-blue-500" />}
            </div>
            {confirmed ? (
              <>
                <h2 className="mb-2 text-2xl font-bold">Booking Confirmed!</h2>
                <p className="mb-1 text-sm text-gray-500">Your PNR is <span className="text-xl font-bold text-brand-indigo">{booking.pnr}</span></p>
                <p className="mb-6 text-sm text-gray-500">Amount: ${(booking.amount_cents / 100).toFixed(2)} {booking.currency}</p>
              </>
            ) : (
              <>
                <h2 className="mb-2 text-2xl font-bold">Booking Created</h2>
                <p className="mb-1 text-sm text-gray-500">Your PNR is <span className="text-xl font-bold text-brand-indigo">{booking.pnr}</span></p>
                <p className="mb-1 text-sm text-gray-500">Status: <span className="font-medium">{booking.status}</span></p>
                <p className="mb-6 text-sm text-gray-500">Amount: ${(booking.amount_cents / 100).toFixed(2)} {booking.currency}</p>
              </>
            )}
            {!confirmed && stripeConfigured && booking.payment_client_secret ? (
              <div className="mb-3">
                <StripePaymentForm
                  clientSecret={booking.payment_client_secret}
                  amountCents={booking.amount_cents}
                  currency={booking.currency}
                  returnUrl={`${window.location.origin}${window.location.pathname}?pnr=${booking.pnr}&payment=return`}
                  onSuccess={handleConfirm}
                />
              </div>
            ) : (
              !confirmed && (
                <Button className="btn-gradient w-full mb-3" onClick={handleConfirm} disabled={busy}>
                  {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "Confirm Payment"}
                </Button>
              )
            )}
            <div className="flex gap-3">
              <Button variant="outline" className="flex-1" onClick={() => router.push(`/dashboard/bookings?pnr=${booking.pnr}`)}>View Booking</Button>
              <Button variant="outline" className="flex-1" onClick={() => router.push("/search")}>Search Again</Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
