import localUnitTest from './LocalUnit.test';
import networkConnectionTest from './NetworkConnection.test';

export default function testsuite() {
  localUnitTest();
  networkConnectionTest();
}