/**
 * Copyright 2004-present Facebook. All Rights Reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 *
 * @flow
 * @format
 */

if (!process.env.NODE_ENV) {
  process.env.BABEL_ENV = 'development';
  process.env.NODE_ENV = 'development';
} else {
  process.env.BABEL_ENV = process.env.NODE_ENV;
}

const {DEV_MODE, LOG_FORMAT, LOG_LEVEL} = require('../server/config');

// This must be done before any module imports to configure
// logging correctly
const logging = require('@fbcnms/logging');
logging.configure({
  LOG_FORMAT,
  LOG_LEVEL,
});

const {
  middleware: fbcExpressMiddleware,
} = require('@fbcnms/express-middleware');
const connectSession = require('connect-session-sequelize');
const express = require('express');
const passport = require('passport');
const fbcPassport = require('@fbcnms/auth/passport').default;
const paths = require('fbcnms-webpack-config/paths');
const session = require('express-session');
const {sequelize, User} = require('../server/models');
const {runMigrations} = require('./runMigrations');
const {access, configureAccess} = require('@fbcnms/auth/access');
const {
  AccessRoles: {USER},
} = require('@fbcnms/auth/roles');

const logger = logging.getLogger(module);

const port = process.env.PORT || 80;

const app = express();

// Create Sequelize Store
const SessionStore = connectSession(session.Store);
const sequelizeSessionStore = new SessionStore({db: sequelize});

// FBC express initialization
app.use(
  fbcExpressMiddleware(app, {
    devMode: DEV_MODE,
    distPath: paths.distPath,
    sessionStore: sequelizeSessionStore,
    sessionToken:
      process.env.SESSION_TOKEN || 'ikbhlkrllnkjikrbkfelltujnnhdlcvb',
    devWebpackConfig: require('../config/webpack.config.js'),
  }),
);

// Initialize Passport (must be after session)
fbcPassport.use({UserModel: User});
app.use(passport.initialize());
app.use(passport.session());

// Restrict all endpoints to at least USER level
app.use(
  configureAccess({
    loginUrl: '/nms/user/login',
  }),
);
app.use(access(USER));

// Views
app.set('views', './views');
app.set('view engine', 'pug');

// Routes
app.use('/nms', require('../server/nms/routes'));

// For backward compat, a health check
app.use('/user/login', (req, res) => {
  res.send('OK');
});

app.get('/', (req, res) => {
  res.redirect('/nms');
});

app.get('*', (req, res) => {
  res.status(404);
  res.send('Not Found');
});

(async function main() {
  await runMigrations();

  app.listen(port, '', err => {
    if (err) {
      logger.error(err);
    }
    if (DEV_MODE) {
      logger.info(`Development server started on port ${port}`);
    } else {
      logger.info(`Production server started on port ${port}`);
    }
  });
})().catch(error => {
  logger.error(error);
});
