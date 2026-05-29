import { createRouter, createWebHistory } from 'vue-router'
import Login from './views/Login.vue'
import AppLayout from './components/AppLayout.vue'
import Home from './views/Home.vue'
import ProjectsIndex from './views/projects/Index.vue'
import ProjectsNew from './views/projects/New.vue'
import ProjectsShow from './views/projects/Show.vue'
import ProjectsEdit from './views/projects/Edit.vue'
import ModelsIndex from './views/models/Index.vue'
import ModelsNew from './views/models/New.vue'
import ModelsEdit from './views/models/Edit.vue'
import CredentialsIndex from './views/credentials/Index.vue'
import CredentialsNew from './views/credentials/New.vue'
import CredentialsEdit from './views/credentials/Edit.vue'
import TasksIndex from './views/tasks/Index.vue'
import TasksShow from './views/tasks/Show.vue'
import Settings from './views/Settings.vue'
import SensitiveWordsIndex from './views/sensitive-words/Index.vue'
import UsersIndex from './views/users/Index.vue'
import UsersNew from './views/users/New.vue'
import UsersEdit from './views/users/Edit.vue'
import Docs from './views/Docs.vue'
import ScanSchedulesIndex from './views/scan-schedules/Index.vue'
import NotifyIndex from './views/notify/Index.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/login', component: Login, meta: { public: true } },
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: 'home',                    component: Home },
        { path: 'projects',                component: ProjectsIndex },
        { path: 'projects/new',            component: ProjectsNew },
        { path: 'projects/:id',            component: ProjectsShow },
        { path: 'projects/:id/edit',       component: ProjectsEdit },
        { path: 'tasks',                   component: TasksIndex },
        { path: 'tasks/:id',               component: TasksShow },
        // Admin-only routes
        { path: 'scan-schedules',          component: ScanSchedulesIndex },
        { path: 'notify',                  component: NotifyIndex },
        { path: 'models',                  component: ModelsIndex,          meta: { adminOnly: true } },
        { path: 'models/new',              component: ModelsNew,            meta: { adminOnly: true } },
        { path: 'models/:id/edit',         component: ModelsEdit,           meta: { adminOnly: true } },
        { path: 'credentials',             component: CredentialsIndex },
        { path: 'credentials/new',         component: CredentialsNew },
        { path: 'credentials/:id/edit',    component: CredentialsEdit },
        { path: 'sensitive-words',         component: SensitiveWordsIndex,  meta: { adminOnly: true } },
        { path: 'docs',                    component: Docs },
        { path: 'settings',                component: Settings,             meta: { adminOnly: true } },
        { path: 'users',                   component: UsersIndex,           meta: { adminOnly: true } },
        { path: 'users/new',              component: UsersNew,              meta: { adminOnly: true } },
        { path: 'users/:id/edit',         component: UsersEdit,             meta: { adminOnly: true } },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) return '/login'
  if (to.path === '/login' && token) return '/home'
  if (to.meta.adminOnly) {
    const role = localStorage.getItem('role') || ''
    if (role !== 'admin' && role !== 'super_admin') return '/home'
  }
})

export default router
