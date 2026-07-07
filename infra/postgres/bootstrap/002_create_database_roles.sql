-- Auth Service
CREATE ROLE auth_user
LOGIN               -- adding LOGIN makes it a database user that your Go service can authenticate as.
PASSWORD 'auth_password';

-- Core Service
CREATE ROLE core_user
LOGIN
PASSWORD 'core_password';

-- Merchant Service
CREATE ROLE merchant_user
LOGIN
PASSWORD 'merchant_password';

-- Rewards Service
CREATE ROLE rewards_user
LOGIN
PASSWORD 'rewards_password';

-- Automation Service
CREATE ROLE automation_user
LOGIN
PASSWORD 'automation_password';