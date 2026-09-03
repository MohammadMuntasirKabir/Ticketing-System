const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

interface ApiOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
}

interface Envelope<T> {
  data: T;
  meta?: { total?: number; currency?: string };
}

export interface Airline {
  id: number;
  code: string;
  name: string;
  country?: string;
}

export interface Airport {
  id: number;
  code: string;
  name: string;
  city: string;
  country: string;
}

export interface FareOption {
  fare_class_code: string;
  cabin: string;
  available: number;
  price: number;
  taxes: number;
  total: number;
}

export interface SearchResult {
  flight_id: number;
  airline: Airline;
  flight_number: string;
  origin: Airport;
  destination: Airport;
  departure: string;
  arrival: string;
  duration_minutes: number;
  aircraft: string;
  fares: FareOption[];
}

export interface SeatMapData {
  aircraft: string;
  cabins: { code: string; name: string; rows: { from: number; to: number }; cols: string[] }[];
  seats: {
    id: number;
    row: number;
    col: string;
    position: string;
    cabin: string;
    is_available: boolean;
    is_emergency_exit: boolean;
    is_extra_legroom: boolean;
  }[];
  legend: Record<string, { color: string; label: string }>;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface PriceQuote {
  quote_id: string;
  flight_id: number;
  fare_class_code: string;
  price: number;
  taxes: number;
  total: number;
  currency: string;
  expires_at: string;
}

export interface PassengerInput {
  title: string;
  first_name: string;
  last_name: string;
  date_of_birth: string;
  nationality: string;
  passport_number: string;
  passport_expiry: string;
}

export interface CreateBookingResponse {
  pnr: string;
  pnr_id: number;
  status: string;
  payment_intent_id: string;
  payment_client_secret: string;
  amount_cents: number;
  currency: string;
  hold_expires_at: string;
}

export interface BookingDetailEntry {
  id: number;
  flight_number: string;
  fare_class: string;
  seat: string;
  status: string;
}

export interface BookingDetail {
  pnr: string;
  status: string;
  total_cents: number;
  currency: string;
  created_at: string;
  bookings: BookingDetailEntry[];
}

export interface CheckInResult {
  id: number;
  pnr_id: number;
  booking_id: number;
  checked_in_at: string;
  is_online: boolean;
  baggage_count: number;
  baggage_weight: number;
}

export interface BoardingPass {
  id: number;
  check_in_id: number;
  passenger_id: number;
  flight_number: string;
  origin: string;
  destination: string;
  seat_number: string;
  boarding_group: string;
  sequence_number: number;
  gate: string;
  boarding_time: string;
  qr_code_data: string;
  issued_at: string;
}

export interface FlightInstance {
  id: number;
  schedule_id: number;
  departure_date: string;
  departure_time: string;
  arrival_time: string;
  status: string;
  delay_minutes: number;
}

export interface AdminBookingRow {
  id: number;
  pnr_id: number;
  flight_instance_id: number;
  fare_class_id: number;
  status: string;
  total_price: number;
  currency: string;
}

export interface AdminUser {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  phone?: string | null;
  role: string;
  status: string;
}

export interface LoadFactorReport {
  flight_instance_id: number;
  flight_number: string;
  departure: string;
  capacity: number;
  sold: number;
  load_factor: number;
}

export interface RevenueReport {
  flight_instance_id: number;
  flight_number: string;
  departure: string;
  bookings: number;
  revenue: number;
}

export interface PeakTimesReport {
  hour: number;
  bookings: number;
}

async function apiClient<T>(path: string, opts: ApiOptions = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: opts.method || "GET",
    headers: { "Content-Type": "application/json", ...opts.headers },
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }

  return res.json();
}

function authHeaders(token?: string): Record<string, string> {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export const flightApi = {
  search: (params: Record<string, string>) => {
    const qs = new URLSearchParams(params).toString();
    return apiClient<Envelope<SearchResult[]>>(`/flights/search?${qs}`);
  },
  getDetail: (id: number) => apiClient<SearchResult>(`/flights/${id}`),
  getSeatMap: (id: number) => apiClient<SeatMapData>(`/flights/${id}/seat-map`),
  listAirports: (q?: string) =>
    apiClient<Envelope<Airport[]>>(`/airports${q ? `?q=${q}` : ""}`),
};

export const bookingApi = {
  getPrice: (data: unknown) => apiClient<PriceQuote>("/bookings/price", { method: "POST", body: data }),
  create: (data: unknown) => apiClient<CreateBookingResponse>("/bookings", { method: "POST", body: data }),
  confirm: (pnr: string) =>
    apiClient<{ pnr: string; status: string }>(`/bookings/${pnr}/confirm`, { method: "POST" }),
  getByPNR: (pnr: string) => apiClient<BookingDetail>(`/bookings/${pnr}`),
  selectSeats: (pnr: string, seatNumber: string) =>
    apiClient<{ booking_id: number; seat_id?: number | null; seat: string }>(
      `/bookings/${pnr}/seats`,
      { method: "PUT", body: { seat_number: seatNumber } }
    ),
  cancel: (pnr: string) =>
    apiClient<{ pnr: string; status: string }>(`/bookings/${pnr}`, { method: "DELETE" }),
};

export const paymentApi = {
  createIntent: (data: unknown) => apiClient<Envelope<unknown>>("/payments/create-intent", { method: "POST", body: data }),
};

export const authApi = {
  register: (data: unknown) => apiClient<AuthResponse>("/auth/register", { method: "POST", body: data }),
  login: (data: unknown) => apiClient<AuthResponse>("/auth/login", { method: "POST", body: data }),
  refresh: (data: unknown) => apiClient<AuthResponse>("/auth/refresh", { method: "POST", body: data }),
};

export const checkinApi = {
  initiate: (pnr: string) =>
    apiClient<CheckInResult>(`/check-in/${pnr}`, { method: "POST" }),
  assignSeat: (pnr: string, seatNumber: string) =>
    apiClient<BoardingPass>(`/check-in/${pnr}/assign-seat`, { method: "PUT", body: { seat_number: seatNumber } }),
  boardingPass: (pnr: string) => apiClient<BoardingPass>(`/check-in/${pnr}/boarding-pass`),
};

export const adminApi = {
  flights: {
    list: (token?: string) =>
      apiClient<Envelope<FlightInstance[]>>("/admin/flights", { headers: authHeaders(token) }),
    create: (data: unknown, token?: string) =>
      apiClient<{ id: number }>("/admin/flights", { method: "POST", body: data, headers: authHeaders(token) }),
    update: (id: number, data: unknown, token?: string) =>
      apiClient<{ status: string }>(`/admin/flights/${id}`, { method: "PUT", body: data, headers: authHeaders(token) }),
    delete: (id: number, token?: string) =>
      apiClient<{ status: string }>(`/admin/flights/${id}`, { method: "DELETE", headers: authHeaders(token) }),
  },
  bookings: {
    list: (token?: string) =>
      apiClient<Envelope<AdminBookingRow[]>>("/admin/bookings", { headers: authHeaders(token) }),
  },
  users: {
    list: (token?: string) =>
      apiClient<Envelope<AdminUser[]>>("/admin/users", { headers: authHeaders(token) }),
    update: (id: number, data: unknown, token?: string) =>
      apiClient<{ status: string }>(`/admin/users/${id}`, { method: "PUT", body: data, headers: authHeaders(token) }),
  },
  reports: {
    loadFactor: (token?: string) =>
      apiClient<LoadFactorReport[]>("/admin/reports/load-factor", { headers: authHeaders(token) }),
    revenue: (token?: string) =>
      apiClient<RevenueReport[]>("/admin/reports/revenue", { headers: authHeaders(token) }),
    peakTimes: (token?: string) =>
      apiClient<PeakTimesReport[]>("/admin/reports/peak-times", { headers: authHeaders(token) }),
  },
};
