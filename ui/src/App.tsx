import {
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
  Outlet,
  Navigate,
  redirect,
} from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from './context/AuthContext';
import Login from './pages/Login';
import AdminLayout from './components/admin/AdminLayout';
import StreamsPage from './pages/admin/StreamsPage';
import OverviewPage from './pages/admin/OverviewPage';
import ModerationPage from './pages/admin/ModerationPage';
import FederationPage from './pages/admin/FederationPage';

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 1 } } });

const rootRoute = createRootRoute({ component: Outlet });

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin/ui/login',
  component: Login,
});

const adminRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin/ui',
  beforeLoad: () => {
    if (!localStorage.getItem('adminKey')) {
      throw redirect({ to: '/admin/ui/login' });
    }
  },
  component: AdminLayout,
});

const adminIndexRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: '/',
  component: () => <Navigate to="/admin/ui/streams" />,
});

const streamsRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: '/streams',
  component: StreamsPage,
});

const overviewRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: '/overview',
  component: OverviewPage,
});

const moderationRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: '/moderation',
  component: ModerationPage,
});

const federationRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: '/federation',
  component: FederationPage,
});

const catchAllRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '*',
  component: () => <Navigate to="/admin/ui/streams" />,
});

const routeTree = rootRoute.addChildren([
  loginRoute,
  adminRoute.addChildren([adminIndexRoute, streamsRoute, overviewRoute, moderationRoute, federationRoute]),
  catchAllRoute,
]);

const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export default function App() {
  return (
    <AuthProvider>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </AuthProvider>
  );
}
