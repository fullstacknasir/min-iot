import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login.component';
import { DeviceListComponent } from './pages/devices/device-list.component';
import { DeviceDetailComponent } from './pages/device-detail/device-detail.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { AlertsComponent } from './pages/alerts/alerts.component';
import { RulesComponent } from './pages/rules/rules.component';
import { AdminDevicesComponent } from './pages/admin/admin-devices/admin-devices.component';
import { AdminUsersComponent } from './pages/admin/admin-users/admin-users.component';
import { ProfileComponent } from './pages/profile/profile.component';
import { WidgetLibraryComponent } from './pages/widget-library/widget-library.component';
import { WidgetEditorComponent } from './pages/widget-editor/widget-editor.component';
import { authGuard } from './core/guards/auth.guard';
import { adminGuard } from './core/guards/admin.guard';

export const routes: Routes = [
  { path: '', redirectTo: 'devices', pathMatch: 'full' },
  { path: 'login', component: LoginComponent },
  { path: 'devices', component: DeviceListComponent, canActivate: [authGuard] },
  { path: 'devices/:id', component: DeviceDetailComponent, canActivate: [authGuard] },
  { path: 'dashboards', component: DashboardComponent, canActivate: [authGuard] },
  { path: 'dashboards/:id', component: DashboardComponent, canActivate: [authGuard] },
  { path: 'widgets', component: WidgetLibraryComponent, canActivate: [authGuard] },
  { path: 'widgets/editor', component: WidgetEditorComponent, canActivate: [authGuard] },
  { path: 'widgets/editor/:id', component: WidgetEditorComponent, canActivate: [authGuard] },
  { path: 'alerts', component: AlertsComponent, canActivate: [authGuard] },
  { path: 'rules', component: RulesComponent, canActivate: [authGuard] },
  { path: 'profile', component: ProfileComponent, canActivate: [authGuard] },
  { path: 'admin/devices', component: AdminDevicesComponent, canActivate: [adminGuard] },
  { path: 'admin/users', component: AdminUsersComponent, canActivate: [adminGuard] },
  { path: '**', redirectTo: 'devices' },
];
