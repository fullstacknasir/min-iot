import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login.component';
import { DeviceListComponent } from './pages/devices/device-list.component';
import { DeviceDetailComponent } from './pages/device-detail/device-detail.component';
import { AdminDevicesComponent } from './pages/admin/admin-devices/admin-devices.component';
import { AdminUsersComponent } from './pages/admin/admin-users/admin-users.component';
import { authGuard } from './core/guards/auth.guard';
import { adminGuard } from './core/guards/admin.guard';

export const routes: Routes = [
  { path: '', redirectTo: 'devices', pathMatch: 'full' },
  { path: 'login', component: LoginComponent },
  { path: 'devices', component: DeviceListComponent, canActivate: [authGuard] },
  { path: 'devices/:id', component: DeviceDetailComponent, canActivate: [authGuard] },
  { path: 'admin/devices', component: AdminDevicesComponent, canActivate: [adminGuard] },
  { path: 'admin/users', component: AdminUsersComponent, canActivate: [adminGuard] },
  { path: '**', redirectTo: 'devices' },
];
